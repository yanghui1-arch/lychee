//go:build !windows || !amd64 || !cgo

package app

import (
	"context"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func promptCloseChoice(ctx context.Context) (closeDecision, error) {
	choice, err := wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
		Type: wailsruntime.QuestionDialog, Title: "荔枝 · 关闭选项", Message: "关闭还是最小化？",
		Buttons: []string{"关闭荔枝", "最小化", "取消"}, DefaultButton: "最小化", CancelButton: "取消",
	})
	decision := closeDecision{}
	if choice == "关闭荔枝" {
		decision.action = closeQuit
	} else if choice == "最小化" {
		decision.action = closeMinimize
	}
	return decision, err
}
