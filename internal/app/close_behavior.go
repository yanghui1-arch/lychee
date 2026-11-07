package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type closeBehavior string

const (
	closeAsk      closeBehavior = "ask"
	closeQuit     closeBehavior = "quit"
	closeMinimize closeBehavior = "minimize"
)

type closeDecision struct {
	action   closeBehavior
	remember bool
}

func validCloseBehavior(action closeBehavior) bool {
	return action == closeAsk || action == closeQuit || action == closeMinimize
}

func defaultClosePreferencePath() string {
	directory, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(directory, "lychee", "close-behavior.json")
}

func readCloseBehavior(path string) closeBehavior {
	data, err := os.ReadFile(path)
	var preference struct {
		Action closeBehavior `json:"action"`
	}
	if err != nil || json.Unmarshal(data, &preference) != nil || !validCloseBehavior(preference.Action) {
		return closeAsk
	}
	return preference.Action
}

func writeCloseBehavior(path string, action closeBehavior) error {
	if !validCloseBehavior(action) {
		return errors.New("无效的关闭选项")
	}
	if path == "" {
		return errors.New("无法找到用户配置目录")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".close-behavior-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	data, _ := json.Marshal(struct {
		Action closeBehavior `json:"action"`
	}{action})
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func resolveCloseBehavior(path string, prompt func() (closeDecision, error)) (closeBehavior, error) {
	if remembered := readCloseBehavior(path); remembered != closeAsk {
		return remembered, nil
	}
	decision, err := prompt()
	if err != nil {
		return "", err
	}
	if decision.action == "" { // Cancel, Escape, or the dialog's own close button.
		return "", nil
	}
	if decision.action != closeQuit && decision.action != closeMinimize {
		return "", errors.New("无效的关闭选择")
	}
	if decision.remember {
		if err := writeCloseBehavior(path, decision.action); err != nil {
			return "", fmt.Errorf("无法保存关闭选项: %w", err)
		}
	}
	return decision.action, nil
}

func (s *WindowService) GetCloseBehavior() string {
	return string(readCloseBehavior(s.closePreferencePath))
}

func (s *WindowService) SetCloseBehavior(action string) error {
	return writeCloseBehavior(s.closePreferencePath, closeBehavior(action))
}

// RequestClose is shared by the floating ball, workspace title bar, and Alt+F4.
func (s *WindowService) RequestClose() error {
	if !s.closeMu.TryLock() {
		return nil // A modal choice is already being shown.
	}
	defer s.closeMu.Unlock()
	s.mu.Lock()
	ctx := s.ctx
	s.mu.Unlock()
	if ctx == nil {
		return nil
	}
	action, err := resolveCloseBehavior(s.closePreferencePath, func() (closeDecision, error) {
		return promptCloseChoice(ctx)
	})
	if err != nil {
		wailsruntime.LogError(ctx, err.Error())
		_, _ = wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
			Type: wailsruntime.ErrorDialog, Title: "荔枝", Message: "无法完成关闭操作，请重试；如果勾选了“以后不再提示”，请先取消勾选。",
		})
		return err
	}
	if action == closeQuit || action == closeMinimize {
		s.mu.Lock()
		err := s.saveFloatingPositionLocked()
		s.mu.Unlock()
		if err != nil {
			return err
		}
	}
	switch action {
	case closeQuit:
		s.quitAllowed.Store(true)
		wailsruntime.Quit(ctx)
	case closeMinimize:
		wailsruntime.WindowMinimise(ctx)
	}
	return nil
}

func (s *WindowService) beforeClose(ctx context.Context) bool {
	if s.quitAllowed.Load() {
		return false
	}
	go func() {
		if err := s.RequestClose(); err != nil {
			wailsruntime.LogError(ctx, err.Error())
		}
	}()
	return true
}
