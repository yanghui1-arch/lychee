//go:build windows && amd64

package audio

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestApplicationRoot(t *testing.T) {
	processes := map[uint32]audioProcess{
		1: {1, 0, "explorer.exe"}, 2: {2, 1, "chrome.exe"}, 3: {3, 2, "CHROME.EXE"}, 4: {4, 3, "chrome.exe"},
		5: {5, 1, "chrome.exe"}, 6: {6, 4, "other.exe"},
	}
	if applicationRoot(4, processes) != 2 || applicationRoot(5, processes) != 5 || applicationRoot(6, processes) != 6 {
		t.Fatal("process trees incorrectly grouped")
	}
}

func toneProcess(t *testing.T, frequency int) *exec.Cmd {
	t.Helper()
	var wave bytes.Buffer
	const samples = audioSampleRate / 2
	wave.WriteString("RIFF")
	_ = binary.Write(&wave, binary.LittleEndian, uint32(36+samples*2))
	wave.WriteString("WAVEfmt ")
	_ = binary.Write(&wave, binary.LittleEndian, uint32(16))
	for _, value := range []uint16{1, 1} {
		_ = binary.Write(&wave, binary.LittleEndian, value)
	}
	for _, value := range []uint32{audioSampleRate, audioSampleRate * 2} {
		_ = binary.Write(&wave, binary.LittleEndian, value)
	}
	for _, value := range []uint16{2, 16} {
		_ = binary.Write(&wave, binary.LittleEndian, value)
	}
	wave.WriteString("data")
	_ = binary.Write(&wave, binary.LittleEndian, uint32(samples*2))
	for i := 0; i < samples; i++ {
		_ = binary.Write(&wave, binary.LittleEndian, int16(3276*math.Sin(2*math.Pi*float64(frequency*i)/audioSampleRate)))
	}
	path := filepath.Join(t.TempDir(), "tone.wav")
	if err := os.WriteFile(path, wave.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	script := "$player = New-Object System.Media.SoundPlayer('" + strings.ReplaceAll(path, "'", "''") + "'); $player.PlayLooping(); Write-Output 'ready'; Start-Sleep -Seconds 30"
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	if !bufio.NewScanner(output).Scan() {
		t.Fatal("test tone did not start")
	}
	return cmd
}

func TestApplicationAudioHardware(t *testing.T) {
	if os.Getenv("LYCHEE_AUDIO_HARDWARE_TEST") != "1" {
		t.Skip("set LYCHEE_AUDIO_HARDWARE_TEST=1 to play controlled test tones")
	}
	target := toneProcess(t, 440)
	_ = toneProcess(t, 1000)
	time.Sleep(250 * time.Millisecond)
	sources, err := listAudioSources()
	if err != nil {
		t.Fatal(err)
	}
	var source AudioSource
	for _, item := range sources {
		if item.PID == uint32(target.Process.Pid) {
			source = item
		}
	}
	if source.ID == "" {
		t.Fatal("playing application missing from source list")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	chunks := make(chan []byte, 20)
	ready := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- captureAudio(ctx, source, func() { close(ready) }, func(pcm []byte) {
			select {
			case chunks <- pcm:
			default:
			}
		})
	}()
	select {
	case <-ready:
	case err := <-done:
		t.Fatal(err)
	case <-time.After(12 * time.Second):
		t.Fatal("capture activation timed out")
	}
	var pcm []byte
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	for pcmLevel(pcm) < 0.005 {
		select {
		case pcm = <-chunks:
		case err := <-done:
			t.Fatal(err)
		case <-deadline.C:
			t.Fatal("no target audio captured")
		}
	}
	amplitude := func(frequency float64) float64 {
		var real, imaginary float64
		for i := 0; i < len(pcm)/2; i++ {
			value := float64(int16(binary.LittleEndian.Uint16(pcm[i*2:])))
			angle := 2 * math.Pi * frequency * float64(i) / audioSampleRate
			real += value * math.Cos(angle)
			imaginary += value * math.Sin(angle)
		}
		return math.Hypot(real, imaginary)
	}
	one, other := amplitude(440), amplitude(1000)
	if one < other*5 {
		t.Fatalf("application audio was not isolated: target=%v unrelated=%v", one, other)
	}
	t.Log(fmt.Sprintf("captured %d bytes, level %.4f, target/unrelated amplitude ratio %.1f", len(pcm), pcmLevel(pcm), one/math.Max(1, other)))
	_ = target.Process.Kill()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "已退出") {
			t.Fatalf("unexpected application exit: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("capture did not stop when application exited")
	}
	systemCtx, stopSystem := context.WithCancel(context.Background())
	defer stopSystem()
	systemDone := make(chan error, 1)
	systemChunks := make(chan float64, 20)
	go func() {
		systemDone <- captureAudio(systemCtx, AudioSource{ID: "system", Kind: "system"}, func() {}, func(pcm []byte) {
			select {
			case systemChunks <- pcmLevel(pcm):
			default:
			}
		})
	}()
	systemDeadline := time.NewTimer(5 * time.Second)
	defer systemDeadline.Stop()
	for {
		select {
		case level := <-systemChunks:
			if level < 0.005 {
				continue
			}
			stopSystem()
			select {
			case err := <-systemDone:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("system audio did not stop")
			}
			return
		case err := <-systemDone:
			t.Fatal(err)
		case <-systemDeadline.C:
			t.Fatal("system loopback received no test audio")
		}
	}
}
