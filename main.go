package main

//#include <dirent.h>
//#include <stdlib.h>
import "C"

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"unsafe"
)

func main() {
	root := flag.String("in", ".", "Search within `dir`")
	flag.Parse()
	s := scanner{[]byte(flag.Arg(0)), make(chan string, 8), make(chan error, 8)}
	go s.scanRootDir(*root)
	for {
		select {
		case name := <-s.foundNames:
			os.Stdout.WriteString(name + "\n")
		case err := <-s.errors:
			if err == nil {
				return
			}
			os.Stderr.WriteString(err.Error() + "\n")
		}
	}
}

type scanner struct {
	pattern    []byte
	foundNames chan string
	errors     chan error
}

func (s *scanner) scanRootDir(dir string) {
	s.scanDir(dir)
	close(s.errors)
}

func (s *scanner) scanDir(dir string) {
	cdir := C.CString(dir)
	defer C.free(unsafe.Pointer(cdir))
	d, err := C.opendir(cdir)
	if err != nil {
		s.errors <- err
		return
	}
	defer C.closedir(d)
	for {
		entry, err := C.readdir(d)
		if err != nil {
			s.errors <- err
			return
		}
		if entry == nil {
			return
		}
		name := (*(*[1 << 30]byte)(unsafe.Pointer(&entry.d_name)))[:entry.d_namlen]
		if (len(name) == 0 || name[0] == '.') && (len(name) <= 1 || name[1] == '.') {
			continue
		}
		if bytes.Contains(name, s.pattern) {
			s.foundNames <- filepath.Join(dir, string(name))
		}
		if entry.d_type == C.DT_DIR {
			s.scanDir(filepath.Join(dir, string(name)))
		}
	}
}
