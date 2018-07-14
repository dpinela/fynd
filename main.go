package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	root := flag.String("in", ".", "Search within `dir`")
	flag.Parse()
	s := scanner{flag.Arg(0), make(chan string, 8), make(chan error, 8)}
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
	pattern    string
	foundNames chan string
	errors     chan error
}

func (s *scanner) scanRootDir(dir string) {
	s.scanDir(dir)
	close(s.errors)
}

func (s *scanner) scanDir(dir string) {
	d, err := os.Open(dir)
	if err != nil {
		s.errors <- err
		return
	}
	defer d.Close()
	for {
		entries, err := d.Readdir(200)
		if err != nil {
			if err != io.EOF {
				s.errors <- err
			}
			return
		}
		for _, e := range entries {
			if strings.Contains(e.Name(), s.pattern) {
				s.foundNames <- filepath.Join(dir, e.Name())
			}
			if e.IsDir() {
				s.scanDir(filepath.Join(dir, e.Name()))
			}
		}
	}
}
