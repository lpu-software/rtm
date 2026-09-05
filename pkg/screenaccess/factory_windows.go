//go:build windows

package screenaccess

func NewEngine() (ScreenEngine, error) {
	return NewWindowsScreenEngine()
}
