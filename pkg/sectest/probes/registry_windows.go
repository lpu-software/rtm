//go:build windows

package probes

func (r *ProbeRegistry) registerDarwinProbes() {}

func (r *ProbeRegistry) registerWindowsProbes() {
	r.probes[MethodGDIScreenDC] = NewWinGDIScreenProbe()
	r.probes[MethodGDIWindowDC] = NewWinGDIWindowProbe()
	r.probes[MethodPrintWindowFull] = NewWinPrintWindowProbe(true)
	r.probes[MethodPrintWindowDefault] = NewWinPrintWindowProbe(false)
	r.probes[MethodWindowsGraphicsCapture] = NewWinWGCProbe()
}
