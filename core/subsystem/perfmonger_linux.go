// +build linux

package subsystem

import (
    "os"
    "bufio"
    "fmt"
)

type PlatformHeader LinuxHeader

func NewPlatformHeader() *LinuxHeader {
    header := new(LinuxHeader)
    header.Devices = make(map[string]LinuxDevice)

    header.getDevsParts()

    return header
}

func (header *LinuxHeader) getDevsParts() {
    f, err := os.Open("/proc/diskstats")
    if err != nil {
        panic(err)
    }
    defer f.Close()
    scan := bufio.NewScanner(f)
    for scan.Scan() {
        var major, minor int
        var name string
        c, err := fmt.Sscanf(scan.Text(), "%d %d %s", &major, &minor, &name)
        if err != nil {
            panic(err)
        }
        if c != 3 {
            continue
        }

        header.DevsParts = append(header.DevsParts, name)

        if isDevice(name) {
            header.Devices[name] = LinuxDevice{
                name, getPartitions(name),
            }
        }
    }
}

func isDevice(name string) bool {
    stat, err := os.Stat(fmt.Sprintf("/sys/block/%s", name))
    if err == nil && stat.IsDir() {
        return true
    }

    return false
}

func getPartitions(name string) []string {
    var dir *os.File
    var fis []os.FileInfo
    var err error
    var parts = []string{}

    dir, err = os.Open(fmt.Sprintf("/sys/block/%s", name))
    if err != nil {
        panic(err)
    }
    fis, err = dir.Readdir(0)
    if err != nil {
        panic(err)
    }
    for _, fi := range fis {
        _, err := os.Stat(fmt.Sprintf("/sys/block/%s/%s/stat", name, fi.Name()))
        if err == nil {
            // partition exists
            parts = append(parts, fi.Name())
        }
    }

    return parts
}

