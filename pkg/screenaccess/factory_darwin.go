//go:build darwin

package screenaccess

func NewEngine() (ScreenEngine, error) {
	return NewDarwinScreenEngine()
}
