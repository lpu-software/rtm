package probes

import (
	"runtime"
)

// ProbeRegistry manages the available capture probes on the current system.
type ProbeRegistry struct {
	probes map[CaptureMethod]CaptureProbe
}

// NewProbeRegistry initializes and discovers probes based on the active operating system.
func NewProbeRegistry() *ProbeRegistry {
	r := &ProbeRegistry{
		probes: make(map[CaptureMethod]CaptureProbe),
	}
	r.discoverProbes()
	return r
}

// GetProbe returns the probe associated with a capture method.
func (r *ProbeRegistry) GetProbe(method CaptureMethod) (CaptureProbe, bool) {
	p, ok := r.probes[method]
	return p, ok
}

// GetAllProbes returns all registered probes.
func (r *ProbeRegistry) GetAllProbes() []CaptureProbe {
	list := make([]CaptureProbe, 0, len(r.probes))
	for _, p := range r.probes {
		list = append(list, p)
	}
	return list
}

// GetSupportedMethods returns the list of capture methods available for the current OS.
func (r *ProbeRegistry) GetSupportedMethods() []CaptureMethod {
	methods := make([]CaptureMethod, 0, len(r.probes))
	for m := range r.probes {
		methods = append(methods, m)
	}
	return methods
}

// discoverProbes registers OS-specific probes dynamically.
func (r *ProbeRegistry) discoverProbes() {
	if runtime.GOOS == "darwin" {
		r.registerDarwinProbes()
	} else if runtime.GOOS == "windows" {
		r.registerWindowsProbes()
	}

	// Register generic screenshot probe fallback
	r.registerFallbackProbes()
}
