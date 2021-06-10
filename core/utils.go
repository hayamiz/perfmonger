package core

import (
	"bufio"
	"compress/gzip"
	"io"
)

func NewPerfmongerLogReader(source io.Reader) io.Reader {
	var ret io.Reader
	reader := bufio.NewReader(source)

	magic_numbers, e := reader.Peek(2)
	if e != nil {
		panic(e)
	}

	// check magic number
	if magic_numbers[0] == 0x1f && magic_numbers[1] == 0x8b {
		// gzipped gob input
		ret, e = gzip.NewReader(reader)
		if e != nil {
			panic(e)
		}
	} else {
		// plain gob input
		ret = reader
	}

	return ret
}
