package main

//#include <dirent.h>
//#include <stdlib.h>
import "C"

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"unsafe"
)

func main() {
	root := flag.String("in", ".", "Search within `dir`")
	invert := flag.Bool("not", false, "List files that don't match the pattern")
	flag.Parse()
	re, err := regexp.Compile("(?i)" + flag.Arg(0))
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		return
	}
	s := scanner{
		pattern: re, invert: *invert,
		dirs: make(chan string), foundNames: make(chan string, 8), errors: make(chan error, 8)}
	for i := 0; i < runtime.NumCPU(); i++ {
		go s.work()
	}
	s.wg.Add(1)
	s.dirs <- *root
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()
	for {
		select {
		case name := <-s.foundNames:
			os.Stdout.WriteString(name + "\n")
		case err := <-s.errors:
			os.Stderr.WriteString(err.Error() + "\n")
		case <-done:
			return
		}
	}
}

type scanner struct {
	pattern *regexp.Regexp
	invert  bool

	dirs       chan string
	foundNames chan string
	errors     chan error

	wg sync.WaitGroup
}

func (s *scanner) work() {
	for dir := range s.dirs {
		s.scanDir(dir)
	}
}

func (s *scanner) scanDir(dir string) {
	defer s.wg.Done()
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
		if entry == nil { // EOF
			return
		}
		namep := (*[1 << 30]byte)(unsafe.Pointer(&entry.d_name))
		namlen := bytes.IndexByte((*namep)[:1<<30], 0)
		name := (*namep)[:namlen]
		if (len(name) == 0 || name[0] == '.') && (len(name) <= 1 || name[1] == '.') {
			continue
		}
		if s.pattern.Match(name) == !s.invert {
			s.foundNames <- filepath.Join(dir, string(name))
		}
		if entry.d_type == C.DT_DIR {
			subdir := filepath.Join(dir, string(name))
			s.wg.Add(1)
			select {
			case s.dirs <- subdir:
			default:
				s.scanDir(subdir)
			}
		}
	}
}
