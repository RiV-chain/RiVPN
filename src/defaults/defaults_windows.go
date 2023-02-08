//go:build windows
// +build windows

package defaults

// Sane defaults for the Windows platform. The "default" options may be
// may be replaced by the running configuration.
func getDefaults() platformDefaultParameters {
	return platformDefaultParameters{

		// Configuration (used for meshctl)
		DefaultConfigFile: "C:\\ProgramData\\RiV-mesh\\vpn.conf",
	}
}
