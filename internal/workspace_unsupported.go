//go:build !windows || !amd64

package app

import (
	"context"
	"errors"
)

func (s *WindowService) openWorkspace() error {
	return errors.New("独立工作区暂仅支持 Windows 64 位")
}

func (s *WindowService) watchWorkspaceParent(context.Context, int) {}
