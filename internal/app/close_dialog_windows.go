//go:build windows && amd64 && cgo

package app

/*
#ifndef _WIN32_WINNT
#define _WIN32_WINNT 0x0600
#endif
#include <windows.h>
#include <commctrl.h>
#include <stdint.h>

// Let the Windows SDK handle TASKDIALOGCONFIG's packed ABI and UTF-16 strings.
static HRESULT lycheeCloseDialog(uintptr_t owner, int *button, BOOL *remember) {
    // Load lazily: Go test/binding helper executables have no v6 manifest,
    // whereas the packaged Wails application does. Never make TaskDialog an
    // eager PE import, or those helper executables fail before main starts.
    HMODULE controls = LoadLibraryW(L"comctl32.dll");
    if (!controls) return HRESULT_FROM_WIN32(GetLastError());
    typedef HRESULT (WINAPI *DialogFunction)(const TASKDIALOGCONFIG*, int*, int*, BOOL*);
    DialogFunction showDialog = (DialogFunction)GetProcAddress(controls, "TaskDialogIndirect");
    if (!showDialog) {
        FreeLibrary(controls);
        return HRESULT_FROM_WIN32(ERROR_PROC_NOT_FOUND);
    }
    const TASKDIALOG_BUTTON choices[] = {
        {1001, L"关闭荔枝\n退出应用并停止监听。"},
        {1002, L"最小化\n保留当前状态，继续在后台运行。"}
    };
    TASKDIALOGCONFIG dialog = {0};
    dialog.cbSize = sizeof(dialog);
    dialog.hwndParent = (HWND)owner;
    dialog.dwFlags = TDF_ALLOW_DIALOG_CANCELLATION | TDF_USE_COMMAND_LINKS;
    dialog.dwCommonButtons = TDCBF_CANCEL_BUTTON;
    dialog.pszWindowTitle = L"荔枝 · 关闭选项";
    dialog.pszMainInstruction = L"关闭还是最小化？";
    dialog.cButtons = 2;
    dialog.pButtons = choices;
    dialog.nDefaultButton = 1002;
    dialog.pszVerificationText = L"以后不再提示，记住本次选择";
    dialog.cxWidth = 230;
    HRESULT result = showDialog(&dialog, button, NULL, remember);
    FreeLibrary(controls);
    return result;
}
*/
import "C"

import (
	"context"
	"fmt"
	"runtime"
)

func promptCloseChoice(_ context.Context) (closeDecision, error) {
	owner, err := mainWindowHandle()
	if err != nil {
		return closeDecision{}, err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	var button C.int
	var remember C.BOOL
	status := C.lycheeCloseDialog(C.uintptr_t(owner), &button, &remember)
	if status < 0 {
		return closeDecision{}, fmt.Errorf("关闭选项对话框失败: HRESULT 0x%08x", uint32(status))
	}
	decision := closeDecision{remember: remember != 0}
	switch button {
	case 1001:
		decision.action = closeQuit
	case 1002:
		decision.action = closeMinimize
	}
	return decision, nil
}
