//go:build !darwin && !windows

package probes

func (r *ProbeRegistry) registerDarwinProbes() {}
func (r *ProbeRegistry) registerWindowsProbes() {}
