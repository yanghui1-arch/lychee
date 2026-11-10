//go:build windows && amd64

package audio

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
	"golang.org/x/sys/windows"
)

var activateAudio = windows.NewLazySystemDLL("Mmdevapi.dll").NewProc("ActivateAudioInterfaceAsync")
var completionIID = ole.NewGUID("{41D949AB-9862-444A-80F6-C261334DA5EB}")
var agileIID = ole.NewGUID("{94EA2B94-E9CC-49E0-C0FF-EE64CA8F5B90}")

type activationResult struct {
	client *wca.IAudioClient
	err    error
}
type audioActivation struct {
	mu        sync.Mutex
	abandoned bool
	result    chan activationResult
	pins      runtime.Pinner
}
type audioCompletion struct {
	vtable *[4]uintptr
	refs   int32
}
type audioActivationVariant struct {
	vt       uint16
	reserved [3]uint16
	size     uint32
	padding  uint32
	blob     *[3]uint32
}

var audioActivations sync.Map
var audioCompletionVTable = [4]uintptr{
	syscall.NewCallback(audioCompletionQuery), syscall.NewCallback(audioCompletionAddRef),
	syscall.NewCallback(audioCompletionRelease), syscall.NewCallback(audioCompletionDone),
}

func audioCompletionQuery(this *audioCompletion, iid *ole.GUID, output *unsafe.Pointer) uintptr {
	if iid == nil || output == nil {
		return 0x80004003
	}
	*output = nil
	requested := *iid
	if requested != *ole.IID_IUnknown && requested != *completionIID && requested != *agileIID {
		return 0x80004002
	}
	audioCompletionAddRef(this)
	*output = unsafe.Pointer(this)
	return 0
}

func audioCompletionAddRef(this *audioCompletion) uintptr {
	return uintptr(atomic.AddInt32(&this.refs, 1))
}

func audioCompletionRelease(this *audioCompletion) uintptr {
	remaining := atomic.AddInt32(&this.refs, -1)
	if remaining == 0 {
		if activation, ok := audioActivations.LoadAndDelete(this); ok {
			activation.(*audioActivation).pins.Unpin()
		}
	}
	return uintptr(remaining)
}

func audioCompletionDone(this *audioCompletion, operation *ole.IUnknown) uintptr {
	value, ok := audioActivations.Load(this)
	if !ok {
		return 0x80004005
	}
	activation := value.(*audioActivation)
	var resultCode int32
	var unknown *ole.IUnknown
	vtable := (*[4]uintptr)(unsafe.Pointer(operation.RawVTable))
	hr, _, _ := syscall.SyscallN(vtable[3], uintptr(unsafe.Pointer(operation)), uintptr(unsafe.Pointer(&resultCode)), uintptr(unsafe.Pointer(&unknown)))
	result := activationResult{}
	if int32(hr) < 0 {
		result.err = ole.NewError(uintptr(uint32(hr)))
	} else if resultCode < 0 {
		result.err = ole.NewError(uintptr(uint32(resultCode)))
	} else if unknown == nil {
		result.err = errors.New("Windows 未返回音频采集接口")
	} else {
		result.err = unknown.PutQueryInterface(wca.IID_IAudioClient, &result.client)
	}
	if unknown != nil {
		unknown.Release()
	}
	activation.mu.Lock()
	if activation.abandoned {
		if result.client != nil {
			result.client.Release()
		}
	} else {
		activation.result <- result
	}
	activation.mu.Unlock()
	return 0
}

func activateProcessAudio(ctx context.Context, pid uint32) (*wca.IAudioClient, error) {
	if err := activateAudio.Find(); err != nil {
		return nil, errors.New("当前 Windows 不支持单独监听应用")
	}
	completion := &audioCompletion{vtable: &audioCompletionVTable, refs: 1}
	params := &[3]uint32{1, pid, 0} // PROCESS_LOOPBACK, include target process tree.
	variant := &audioActivationVariant{vt: 65, size: 12, blob: params}
	activation := &audioActivation{result: make(chan activationResult, 1)}
	// Windows retains these pointers until its final COM Release, even after cancellation.
	activation.pins.Pin(completion)
	activation.pins.Pin(params)
	activation.pins.Pin(variant)
	audioActivations.Store(completion, activation)
	defer audioCompletionRelease(completion)
	path, _ := windows.UTF16PtrFromString("VAD\\Process_Loopback")
	var operation *ole.IUnknown
	hr, _, _ := activateAudio.Call(uintptr(unsafe.Pointer(path)), uintptr(unsafe.Pointer(wca.IID_IAudioClient)), uintptr(unsafe.Pointer(variant)), uintptr(unsafe.Pointer(completion)), uintptr(unsafe.Pointer(&operation)))
	if int32(hr) < 0 {
		return nil, ole.NewError(uintptr(uint32(hr)))
	}
	if operation == nil {
		return nil, errors.New("Windows 未返回音频激活任务")
	}
	defer operation.Release()
	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()
	select {
	case result := <-activation.result:
		return result.client, result.err
	case <-ctx.Done():
	case <-timeout.C:
	}
	activation.mu.Lock()
	activation.abandoned = true
	select {
	case result := <-activation.result:
		if result.client != nil {
			result.client.Release()
		}
	default:
	}
	activation.mu.Unlock()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return nil, errors.New("应用音频初始化超时，请重试")
}
