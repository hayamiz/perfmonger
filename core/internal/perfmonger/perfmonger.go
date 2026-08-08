package perfmonger

import (
	"sort"
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
)

type PlatformType int

type CommonHeader struct {
	Platform  PlatformType
	Hostname  string
	StartTime time.Time
	// Meta holds user-supplied key/value metadata injected at record time via
	// the --meta flag. Keys and values are stored decoded (not percent-encoded).
	// The field is optional: older .pgr files encoded before this field existed
	// decode with Meta == nil, and files written with no --meta also leave it
	// nil, so readers must treat a nil/empty map as "no metadata". Being a plain
	// gob field addition, it is forward/backward compatible and requires no
	// format version bump.
	Meta map[string]string
}

// SortedMetaKeys returns the keys of the header's Meta map in ascending order,
// so that metadata can be printed deterministically. It returns an empty slice
// when no metadata is present.
func (h *CommonHeader) SortedMetaKeys() []string {
	keys := make([]string, 0, len(h.Meta))
	for k := range h.Meta {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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

