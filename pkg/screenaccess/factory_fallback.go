//go:build !darwin && !windows

package screenaccess

func NewEngine() (ScreenEngine, error) {
	return NewFallbackScreenEngine()
}
