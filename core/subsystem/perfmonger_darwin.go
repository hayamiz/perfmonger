// +build darwin

package subsystem

type PlatformHeader DarwinHeader

func NewPlatformHeader() *DarwinHeader {
	header := new(DarwinHeader)
	return header
}

func ReadCpuStat(record *StatRecord) error {
	return nil
}

func ReadDiskStats(record *StatRecord, targets *map[string]bool) error {
	return nil
}

func ReadNetStat(record *StatRecord) error {
	return nil
}
