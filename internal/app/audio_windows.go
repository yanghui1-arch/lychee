//go:build windows && amd64

package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
	"golang.org/x/sys/windows"
)

func audioCOM() (func(), error) {
	runtime.LockOSThread()
	err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED)
	if err != nil && err.(*ole.OleError).Code() != 1 {
		runtime.UnlockOSThread()
		return nil, err
	}
	return func() { ole.CoUninitialize(); runtime.UnlockOSThread() }, nil
}

func deviceEnumerator() (*wca.IMMDeviceEnumerator, error) {
	unknown, err := ole.CreateInstance(wca.CLSID_MMDeviceEnumerator, wca.IID_IMMDeviceEnumerator)
	return (*wca.IMMDeviceEnumerator)(unsafe.Pointer(unknown)), err
}

type audioProcess struct {
	pid, parent uint32
	exe         string
}

func processSnapshot() (map[uint32]audioProcess, error) {
	handle, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(handle)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	processes := make(map[uint32]audioProcess)
	for err = windows.Process32First(handle, &entry); err == nil; err = windows.Process32Next(handle, &entry) {
		processes[entry.ProcessID] = audioProcess{entry.ProcessID, entry.ParentProcessID, windows.UTF16ToString(entry.ExeFile[:])}
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return nil, err
	}
	return processes, nil
}

func applicationRoot(pid uint32, processes map[uint32]audioProcess) uint32 {
	seen := make(map[uint32]bool)
	for !seen[pid] {
		seen[pid] = true
		process, ok := processes[pid]
		parent, parentOK := processes[process.parent]
		if !ok || !parentOK || !strings.EqualFold(parent.exe, process.exe) || parent.pid == pid {
			break
		}
		pid = parent.pid
	}
	return pid
}

func processIdentity(handle windows.Handle) (uint64, error) {
	var created, exited, kernel, user windows.Filetime
	err := windows.GetProcessTimes(handle, &created, &exited, &kernel, &user)
	return uint64(created.HighDateTime)<<32 | uint64(created.LowDateTime), err
}

func listAudioSources() ([]AudioSource, error) {
	cleanup, err := audioCOM()
	if err != nil {
		return nil, err
	}
	defer cleanup()
	processes, err := processSnapshot()
	if err != nil {
		return nil, err
	}
	desktop, err := desktopApplicationProcesses()
	if err != nil {
		return nil, err
	}
	seen := desktop.processes
	// Visible desktop apps are selectable before they create an audio session.
	// Also include background apps that are currently rendering audio.
	collectActiveAudioProcesses(seen)
	apps := make(map[uint32]AudioSource)
	windowsDirectory, _ := windows.GetWindowsDirectory()
	for pid := range seen {
		root := applicationRoot(pid, processes)
		if root == uint32(os.Getpid()) || root == 0 {
			continue
		}
		if _, exists := apps[root]; exists {
			continue
		}
		process, ok := processes[root]
		if !ok {
			continue
		}
		if source, ok := applicationAudioSource(root, process.exe, windowsDirectory); ok {
			apps[root] = source
		}
	}
	appList := applicationSourceLabels(apps, desktop.windows, processes)
	return append([]AudioSource{{ID: "system", Label: "全局监听", Kind: "system"}}, appList...), nil
}

func applicationAudioSource(pid uint32, exe, windowsDirectory string) (AudioSource, bool) {
	if !userAudioApplication(exe, "", windowsDirectory) {
		return AudioSource{}, false
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return AudioSource{}, false
	}
	defer windows.CloseHandle(handle)
	var imagePath [32768]uint16
	pathSize := uint32(len(imagePath))
	if windows.QueryFullProcessImageName(handle, 0, &imagePath[0], &pathSize) != nil {
		return AudioSource{}, false
	}
	path := windows.UTF16ToString(imagePath[:pathSize])
	if !userAudioApplication(exe, path, windowsDirectory) {
		return AudioSource{}, false
	}
	created, err := processIdentity(handle)
	if err != nil {
		return AudioSource{}, false
	}
	return AudioSource{
		ID: fmt.Sprintf("app:%d:%d", pid, created), PID: pid, Kind: "application",
		Label: strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe)),
		Icon:  applicationIconFromPath(path),
	}, true
}

// Audio-session discovery supplements desktop windows; unavailable output
// devices must not prevent quiet, visible applications from being listed.
func collectActiveAudioProcesses(seen map[uint32]bool) {
	enumerator, err := deviceEnumerator()
	if err != nil {
		return
	}
	defer enumerator.Release()
	var devices *wca.IMMDeviceCollection
	if enumerator.EnumAudioEndpoints(wca.ERender, wca.DEVICE_STATE_ACTIVE, &devices) != nil {
		return
	}
	defer devices.Release()
	var count uint32
	if devices.GetCount(&count) != nil {
		return
	}
	for i := uint32(0); i < count; i++ {
		var device *wca.IMMDevice
		if devices.Item(i, &device) != nil {
			continue
		}
		collectAudioProcesses(device, seen)
		device.Release()
	}
}

// Keep distinct capture targets, but identify same-named apps by a visible
// window title instead of exposing their internal process IDs.
func applicationSourceLabels(apps map[uint32]AudioSource, desktop []desktopApplicationWindow, processes map[uint32]audioProcess) []AudioSource {
	titles := make(map[uint32]string)
	for _, window := range desktop {
		root := applicationRoot(window.pid, processes)
		if titles[root] == "" && window.title != "" {
			titles[root] = window.title
		}
	}
	counts := make(map[string]int)
	appList := make([]AudioSource, 0, len(apps))
	for _, source := range apps {
		counts[strings.ToLower(source.Label)]++
		appList = append(appList, source)
	}
	sort.Slice(appList, func(i, j int) bool {
		left, right := strings.ToLower(appList[i].Label), strings.ToLower(appList[j].Label)
		if left == right {
			return appList[i].PID < appList[j].PID
		}
		return left < right
	})
	labelCounts := make(map[string]int)
	for i := range appList {
		source := &appList[i]
		if counts[strings.ToLower(source.Label)] > 1 && titles[source.PID] != "" {
			source.Label += " · " + titles[source.PID]
		}
		labelCounts[strings.ToLower(source.Label)]++
	}
	instances := make(map[string]int)
	for i := range appList {
		key := strings.ToLower(appList[i].Label)
		if labelCounts[key] > 1 {
			instances[key]++
			appList[i].Label += fmt.Sprintf(" · 实例 %d", instances[key])
		}
	}
	return appList
}

func collectAudioProcesses(device *wca.IMMDevice, seen map[uint32]bool) {
	var manager *wca.IAudioSessionManager2
	if device.Activate(wca.IID_IAudioSessionManager2, ole.CLSCTX_ALL, nil, &manager) != nil {
		return
	}
	defer manager.Release()
	var sessions *wca.IAudioSessionEnumerator
	if manager.GetSessionEnumerator(&sessions) != nil {
		return
	}
	defer sessions.Release()
	var count int
	if sessions.GetCount(&count) != nil {
		return
	}
	for i := 0; i < count; i++ {
		var session *wca.IAudioSessionControl
		if sessions.GetSession(i, &session) != nil {
			continue
		}
		var control *wca.IAudioSessionControl2
		if session.PutQueryInterface(wca.IID_IAudioSessionControl2, &control) == nil {
			var pid, state uint32
			if control.GetState(&state) == nil && state == 1 && control.GetProcessId(&pid) == nil && pid != 0 {
				seen[pid] = true
			}
			control.Release()
		}
		session.Release()
	}
}

func captureAudio(ctx context.Context, source AudioSource, ready func(), consume func([]byte)) error {
	cleanup, err := audioCOM()
	if err != nil {
		return err
	}
	defer cleanup()
	var client *wca.IAudioClient
	var process windows.Handle
	flags := uint32(wca.AUDCLNT_STREAMFLAGS_EVENTCALLBACK | wca.AUDCLNT_STREAMFLAGS_AUTOCONVERTPCM | wca.AUDCLNT_STREAMFLAGS_SRC_DEFAULT_QUALITY)
	if source.Kind == "application" {
		if windows.RtlGetVersion().BuildNumber < 20348 {
			return errors.New("单独监听应用需要 Windows 11 或 Windows build 20348 及以上")
		}
		parts := strings.Split(source.ID, ":")
		if len(parts) != 3 {
			return errors.New("应用标识无效")
		}
		created, parseErr := strconv.ParseUint(parts[2], 10, 64)
		if parseErr != nil {
			return errors.New("应用标识无效")
		}
		process, err = windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, source.PID)
		if err != nil {
			return errors.New("所选应用已退出，请重新选择")
		}
		defer windows.CloseHandle(process)
		actual, identityErr := processIdentity(process)
		if identityErr != nil || actual != created {
			return errors.New("所选应用已重启，请重新选择")
		}
		client, err = activateProcessAudio(ctx, source.PID)
		flags |= wca.AUDCLNT_STREAMFLAGS_LOOPBACK
	} else if source.Kind == "system" {
		client, err = activateDefaultAudio()
		flags |= wca.AUDCLNT_STREAMFLAGS_LOOPBACK
	} else {
		return errors.New("音频来源无效，请重新选择")
	}
	if err != nil {
		return fmt.Errorf("无法启动音频监听：%w", err)
	}
	defer client.Release()
	format := wca.WAVEFORMATEX{WFormatTag: 1, NChannels: 1, NSamplesPerSec: audioSampleRate, NAvgBytesPerSec: audioSampleRate * 2, NBlockAlign: 2, WBitsPerSample: 16}
	if err = client.Initialize(wca.AUDCLNT_SHAREMODE_SHARED, flags, 0, 0, &format, nil); err != nil {
		return fmt.Errorf("无法初始化音频采集，请检查输出设备：%w", err)
	}
	var capture *wca.IAudioCaptureClient
	if err = client.GetService(wca.IID_IAudioCaptureClient, &capture); err != nil {
		return err
	}
	defer capture.Release()
	event, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(event)
	if err = client.SetEventHandle(uintptr(event)); err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err = client.Start(); err != nil {
		return err
	}
	defer client.Stop()
	ready()
	const chunkBytes = audioSampleRate * 2 / 5
	buffer := make([]byte, 0, chunkBytes*2)
	lastChunk := time.Now()
	for ctx.Err() == nil {
		if process != 0 {
			wait, err := windows.WaitForSingleObject(process, 0)
			if err != nil || wait == windows.WAIT_OBJECT_0 {
				return errors.New("所选应用已退出，监听已停止")
			}
		}
		if _, err = windows.WaitForSingleObject(event, 100); err != nil {
			return err
		}
		for ctx.Err() == nil {
			var frames uint32
			if err = capture.GetNextPacketSize(&frames); err != nil {
				return fmt.Errorf("音频设备已断开或不可用：%w", err)
			}
			if frames == 0 {
				break
			}
			var data *byte
			var packetFlags uint32
			if err = capture.GetBuffer(&data, &frames, &packetFlags, nil, nil); err != nil {
				return err
			}
			if frames > audioSampleRate*2 {
				capture.ReleaseBuffer(frames)
				return errors.New("音频采集返回了异常大小的数据包")
			}
			if packetFlags&wca.AUDCLNT_BUFFERFLAGS_SILENT != 0 || data == nil {
				buffer = append(buffer, make([]byte, int(frames)*2)...)
			} else {
				buffer = append(buffer, unsafe.Slice(data, int(frames)*2)...)
			}
			if err = capture.ReleaseBuffer(frames); err != nil {
				return err
			}
			for len(buffer) >= chunkBytes {
				consume(append([]byte(nil), buffer[:chunkBytes]...))
				buffer = buffer[chunkBytes:]
				lastChunk = time.Now()
			}
		}
		if time.Since(lastChunk) > 250*time.Millisecond {
			// Loopback produces no packets when the output is idle; clear the meter.
			chunk := make([]byte, chunkBytes)
			copy(chunk, buffer)
			buffer = buffer[:0]
			consume(chunk)
			lastChunk = time.Now()
		}
	}
	// A manual stop can land between the 200ms capture blocks. Forward that
	// final partial block before the speech pipeline sends its finish message.
	if len(buffer) > 0 {
		consume(append([]byte(nil), buffer...))
	}
	return nil
}

func activateDefaultAudio() (*wca.IAudioClient, error) {
	enumerator, err := deviceEnumerator()
	if err != nil {
		return nil, err
	}
	defer enumerator.Release()
	var device *wca.IMMDevice
	if err = enumerator.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &device); err != nil {
		return nil, err
	}
	defer device.Release()
	var client *wca.IAudioClient
	err = device.Activate(wca.IID_IAudioClient, ole.CLSCTX_ALL, nil, &client)
	return client, err
}
