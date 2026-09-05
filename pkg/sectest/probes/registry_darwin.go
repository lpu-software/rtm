//go:build darwin

package probes

func (r *ProbeRegistry) registerDarwinProbes() {
	r.probes[MethodScreenCaptureKit] = NewDarwinSCKProbe()
	r.probes[MethodCoreGraphicsWindow] = NewDarwinCGWindowProbe()
	r.probes[MethodCoreGraphicsDisplay] = NewDarwinCGDisplayProbe()
	r.probes[MethodCoreGraphicsRegion] = NewDarwinCGRegionProbe()
}

func (r *ProbeRegistry) registerWindowsProbes() {}
