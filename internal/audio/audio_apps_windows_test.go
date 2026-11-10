//go:build windows && amd64

package audio

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestDesktopApplicationDiscoveryBeforeAudio(t *testing.T) {
	control := filepath.Join(t.TempDir(), "close")
	script := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing;
$form = New-Object System.Windows.Forms.Form;
$form.Text = 'Lychee silent application test'; $form.ShowInTaskbar = $false;
$form.StartPosition = 'Manual'; $form.Location = New-Object System.Drawing.Point(-10000,-10000);
$form.Show(); Write-Output 'ready';
while (-not (Test-Path -LiteralPath '%s')) { [System.Windows.Forms.Application]::DoEvents(); Start-Sleep -Milliseconds 30 };
$form.Close(); Write-Output 'closed'; Start-Sleep -Seconds 20`, strings.ReplaceAll(control, "'", "''"))
	command := exec.Command("powershell.exe", "-STA", "-NoProfile", "-NonInteractive", "-Command", script)
	command.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = command.Process.Kill(); _ = command.Wait() })
	scanner := bufio.NewScanner(output)
	if !scanner.Scan() || scanner.Text() != "ready" {
		t.Fatal("silent application did not start")
	}
	pid := uint32(command.Process.Pid)
	desktop, err := desktopApplicationProcesses()
	if err != nil {
		t.Fatal(err)
	}
	hasTitle := false
	for _, window := range desktop.windows {
		if window.pid == pid && window.title == "Lychee silent application test" {
			hasTitle = true
		}
	}
	if !hasTitle {
		t.Fatal("opened application window title was not detected")
	}
	sources, err := listAudioSources()
	if err != nil {
		t.Fatal(err)
	}
	var source AudioSource
	for _, candidate := range sources {
		if candidate.PID == pid {
			source = candidate
		}
	}
	if source.ID == "" {
		t.Fatal("opened application with no audio session was not detected")
	}
	audio := newAudioService()
	defer audio.StopAudioCapture()
	status, err := audio.StartAudioCapture(source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "listening" || status.Source.PID != pid {
		t.Fatal("silent application cannot be selected for capture")
	}
	audio.StopAudioCapture()
	sources, err = listAudioSources()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, candidate := range sources {
		found = found || candidate.ID == source.ID
	}
	if !found {
		t.Fatal("silent application disappeared while its desktop window was open")
	}
	if err := os.WriteFile(control, []byte("close"), 0600); err != nil {
		t.Fatal(err)
	}
	if !scanner.Scan() || scanner.Text() != "closed" {
		t.Fatal("test window did not close")
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		sources, err = listAudioSources()
		if err != nil {
			t.Fatal(err)
		}
		found = false
		for _, candidate := range sources {
			found = found || candidate.ID == source.ID
		}
		if !found {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("closed, silent desktop window remained in the source list")
}

func TestApplicationSourceLabels(t *testing.T) {
	processes := map[uint32]audioProcess{
		101: {pid: 101, exe: "chrome.exe"},
		102: {pid: 102, parent: 101, exe: "chrome.exe"},
		201: {pid: 201, exe: "chrome.exe"},
	}
	apps := map[uint32]AudioSource{
		101: {ID: "app:101:1", PID: 101, Label: "chrome", Kind: "application", Icon: "first-icon"},
		201: {ID: "app:201:2", PID: 201, Label: "chrome", Kind: "application", Icon: "second-icon"},
		301: {ID: "app:301:3", PID: 301, Label: "WeMeetApp", Kind: "application"},
	}
	tests := []struct {
		name    string
		windows []desktopApplicationWindow
		labels  []string
	}{
		{"different titles", []desktopApplicationWindow{{102, "文档 - Google Chrome"}, {101, "另一窗口"}, {201, "视频 - Google Chrome"}}, []string{"chrome · 文档 - Google Chrome", "chrome · 视频 - Google Chrome", "WeMeetApp"}},
		{"same titles", []desktopApplicationWindow{{101, "新标签页"}, {201, "新标签页"}}, []string{"chrome · 新标签页 · 实例 1", "chrome · 新标签页 · 实例 2", "WeMeetApp"}},
		{"no titles", nil, []string{"chrome · 实例 1", "chrome · 实例 2", "WeMeetApp"}},
		{"one title", []desktopApplicationWindow{{201, "视频"}}, []string{"chrome", "chrome · 视频", "WeMeetApp"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for attempt := 0; attempt < 5; attempt++ {
				sources := applicationSourceLabels(apps, test.windows, processes)
				if len(sources) != len(apps) {
					t.Fatal("independent capture targets were merged or duplicated")
				}
				for i, source := range sources {
					if source.Label != test.labels[i] {
						t.Errorf("label %d = %q, want %q", i, source.Label, test.labels[i])
					}
					original := apps[source.PID]
					if source.ID != original.ID || source.Icon != original.Icon || source.Kind != original.Kind {
						t.Fatal("display label changed the capture target or its icon")
					}
				}
			}
		})
	}
}

func TestUserAudioApplicationFilter(t *testing.T) {
	tests := []struct {
		exe, path string
		want      bool
	}{
		{"ApplicationFrameHost.exe", `C:\Windows\System32\ApplicationFrameHost.exe`, false},
		{"TEXTINPUTHOST.EXE", `C:\Windows\SystemApps\MicrosoftWindows.Client.CBS\TextInputHost.exe`, false},
		{"RtkUWP.exe", `C:\Program Files\WindowsApps\RealtekAudioControl\RtkUWP.exe`, false},
		{"explorer.exe", `C:\Windows\explorer.exe`, false},
		{"OtherShellComponent.exe", `c:\WINDOWS\SYSTEMAPPS\SomePackage\OtherShellComponent.exe`, false},
		{"control.exe", `C:\Windows\ImmersiveControlPanel\control.exe`, false},
		{"chrome.exe", `C:\Program Files\Google\Chrome\Application\chrome.exe`, true},
		{"WeMeetApp.exe", `D:\Apps\TencentMeeting\WeMeetApp.exe`, true},
		{"Code.exe", `C:\Users\user\AppData\Local\Programs\Microsoft VS Code\Code.exe`, true},
		{"ChatGPT.exe", `C:\Program Files\WindowsApps\OpenAI.ChatGPT\ChatGPT.exe`, true},
		{"ms-teams.exe", `C:\Program Files\WindowsApps\MSTeams\ms-teams.exe`, true},
		{"powershell.exe", `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`, true},
		{"player.exe", `C:\Windows\SystemAppsCustom\player.exe`, true},
	}
	for _, test := range tests {
		t.Run(test.exe+test.path, func(t *testing.T) {
			if got := userAudioApplication(test.exe, test.path, `C:\Windows`); got != test.want {
				t.Errorf("userAudioApplication(%q, %q) = %v, want %v", test.exe, test.path, got, test.want)
			}
		})
	}
}

func TestAudioSourcesExcludeSystemComponents(t *testing.T) {
	sources, err := listAudioSources()
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) < 1 || sources[0].ID != "system" || sources[0].Label != "全局监听" {
		t.Fatal("global listening must be the first source")
	}
	for _, source := range sources {
		if source.ID == "microphone" || source.Kind == "microphone" {
			t.Fatal("microphone appeared in source list")
		}
		if source.Kind != "application" {
			continue
		}
		exe := strings.Split(source.Label, " · ")[0] + ".exe"
		if !userAudioApplication(exe, "", "") {
			t.Errorf("system component appeared in source list: %s", source.Label)
		}
	}
}
