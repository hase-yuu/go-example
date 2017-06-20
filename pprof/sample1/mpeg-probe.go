package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pkg/profile"
)

/**
* Mp4 box parser
* Usage: go run mpeg-probe movie.mp4
 */
func main() {
	defer profile.Start(profile.CPUProfile, profile.ProfilePath("./sample.prof")).Stop()

	flag.Parse()
	fileName := flag.Arg(0)
	if fileName == "" {
		os.Exit(1)
		return
	}

	if err := Probe(fileName); err != nil {
		return
	}
}

type header struct {
	Size uint32
	Name string
}

type atom struct {
	Header   *header
	Data     []byte
	Position int64
	Depth    int
}

const (
	headerSize = 8
)

// Probe mpeg atom box
func Probe(in string) error {
	r, err := os.Open(in)
	if err != nil {
		return err
	}
	defer r.Close()

	for {
		a, err := readAtom(r)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		fmt.Printf("[%s] size=%d\n", a.Header.Name, a.Header.Size)

		if err := a.children(); err != nil {
			return err
		}

	}
	return nil
}

func readAtom(r io.Reader) (*atom, error) {
	a := &atom{}
	if err := readHeader(a, r); err != nil {
		return nil, err
	}

	if err := readBody(a, r); err != nil {
		return nil, err
	}

	return a, nil
}

func readHeader(a *atom, r io.Reader) error {
	buf := make([]byte, headerSize)
	if _, err := r.Read(buf); err != nil {
		return err
	}
	a.Header = &header{
		Name: string(buf[4:8]),
		Size: binary.BigEndian.Uint32(buf[0:4]),
	}
	return nil
}

func readBody(a *atom, r io.Reader) error {
	buf := bytes.NewBuffer(nil)
	if _, err := io.CopyN(buf, r, int64(a.Header.Size-headerSize)); err != nil {
		return err
	}
	a.Data = buf.Bytes()
	a.Position = int64(a.Header.Size)
	return nil
}

func discardBody(h *header, r io.Reader) error {
	if _, err := io.CopyN(ioutil.Discard, r, int64(h.Size-headerSize)); err != nil {
		return err
	}
	return nil
}

var isobmff = map[string]bool{
	"ftyp": false,
	"pdin": false,
	"moov": true,
	"mvhd": false,
	"trak": true,
	"tkhd": false,
	"tref": false,
	"edts": false,
	"elst": false,
	"mdia": true,
	"mdhd": false,
	"hdlr": false,
	"minf": true,
	"vmhd": false,
	"smhd": false,
	"hmhd": false,
	"nmhd": false,
	"dinf": true,
	"dref": false,
	"stbl": true,
	"stsd": false,
	"stts": false,
	"ctts": false,
	"stsc": false,
	"stsz": false,
	"stz2": false,
	"stco": false,
	"co64": false,
	"stss": false,
	"stsh": false,
	"padb": false,
	"stdp": false,
	"sdtp": false,
	"sbgp": false,
	"sgpd": false,
	"subs": false,
	"mvex": true,
	"mehd": false,
	"trex": false,
	"ipmc": true,

	"moof": true,
	"mfhd": false,
	"traf": true,
	"trun": false,
	"mfra": true,
	"tfra": false,
	"mfro": false,
	"mdat": false,
	"free": false,
	"skip": true,
}

func hasChild(htype string) bool {
	if ok, exist := isobmff[htype]; ok && exist {
		return true
	}
	return false
}

func (a *atom) children() error {
	if ok := hasChild(a.Header.Name); !ok {
		return nil
	}
	a.Depth++
	buf := bytes.NewBuffer(a.Data)
	var pos int64
	for {
		child, err := readAtom(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if child == nil {
			break
		}
		child.Depth = a.Depth
		pos += child.Position
		child.Position = pos + a.Position
		fmt.Printf("%s[%s] size=%d\n", strings.Repeat("\t", a.Depth), child.Header.Name, child.Header.Size)

		if err := child.children(); err != nil {
			return err
		}
	}
	return nil
}
