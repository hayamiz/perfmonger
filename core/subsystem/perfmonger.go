package subsystem

import (
	"time"
)

/*

PerfMonger binary format:

1. Common header
  * platform type tag
  * host info
  * timestamp
  * ...
2. Platform-dependent header
  * List of devices
  * List of NICs
  * List of IRQs
  * CPU topology
  * ...
3. Record section
  * Platform-dependent record data stream

*/

//
// Common header
//

const (
	Linux  = 1
	Darwin = 2
)

type PlatformType int

type CommonHeader struct {
	Platform  PlatformType
	Hostname  string
	StartTime time.Time
}

//
// Platform-dependent header
//

type LinuxDevice struct {
	Name  string
	Parts []string
}

type LinuxHeader struct {
	Devices   map[string]LinuxDevice
	DevsParts []string
}

type DarwinHeader struct {
	DevsParts []string
}
