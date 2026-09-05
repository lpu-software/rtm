//go:build !darwin && !windows

package testapp

import (
	"fmt"
	"image"
)

// StubWindowBridge implements NativeWindowBridge on unsupported or headless platforms.
type StubWindowBridge struct {
	controller *AppController
	mode       ProtectionMode
}

// NewStubWindowBridge initializes a headless window stub.
func NewStubWindowBridge(ctrl *AppController) (*StubWindowBridge, error) {
	return &StubWindowBridge{
		controller: ctrl,
		mode:       ModeNormal,
	}, nil
}

func (b *StubWindowBridge) ApplyProtectionMode(mode ProtectionMode) error {
	b.mode = mode
	return nil
}

func (b *StubWindowBridge) UpdateDisplay(img *image.RGBA) {}

func (b *StubWindowBridge) GetWindowID() string {
	return "STUB-WIN-0x1"
}

func (b *StubWindowBridge) GetOSProtectionState() (string, bool) {
	if b.mode == ModeOSExclusion || b.mode == ModeCombined {
		return "OS_EXCLUSION_ACTIVE (Simulated)", true
	}
	return "NO_OS_PROTECTION", false
}

func (b *StubWindowBridge) Close() error {
	return nil
}
