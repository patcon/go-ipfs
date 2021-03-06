package util

import (
	"errors"
	"io"
	"math/rand"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	ds "github.com/jbenet/go-ipfs/Godeps/_workspace/src/github.com/jbenet/datastore.go"
)

func init() {
	SetupLogging()
}

// Debug is a global flag for debugging.
var Debug bool

// ErrNotImplemented signifies a function has not been implemented yet.
var ErrNotImplemented = errors.New("Error: not implemented yet.")

// ErrTimeout implies that a timeout has been triggered
var ErrTimeout = errors.New("Error: Call timed out.")

// ErrSeErrSearchIncomplete implies that a search type operation didnt
// find the expected node, but did find 'a' node.
var ErrSearchIncomplete = errors.New("Error: Search Incomplete.")

// ErrNotFound is returned when a search fails to find anything
var ErrNotFound = ds.ErrNotFound

// TildeExpansion expands a filename, which may begin with a tilde.
func TildeExpansion(filename string) (string, error) {
	if strings.HasPrefix(filename, "~/") {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}

		dir := usr.HomeDir + "/"
		filename = strings.Replace(filename, "~/", dir, 1)
	}
	return filename, nil
}

// ExpandPathnames takes a set of paths and turns them into absolute paths
func ExpandPathnames(paths []string) ([]string, error) {
	var out []string
	for _, p := range paths {
		abspath, err := filepath.Abs(p)
		if err != nil {
			return nil, err
		}
		out = append(out, abspath)
	}
	return out, nil
}

// byteChanReader wraps a byte chan in a reader
type byteChanReader struct {
	in  chan []byte
	buf []byte
}

func NewByteChanReader(in chan []byte) io.Reader {
	return &byteChanReader{in: in}
}

func (bcr *byteChanReader) Read(b []byte) (int, error) {
	if len(bcr.buf) == 0 {
		data, ok := <-bcr.in
		if !ok {
			return 0, io.EOF
		}
		bcr.buf = data
	}

	if len(bcr.buf) >= len(b) {
		copy(b, bcr.buf)
		bcr.buf = bcr.buf[len(b):]
		return len(b), nil
	}

	copy(b, bcr.buf)
	b = b[len(bcr.buf):]
	totread := len(bcr.buf)

	for data := range bcr.in {
		if len(data) > len(b) {
			totread += len(b)
			copy(b, data[:len(b)])
			bcr.buf = data[len(b):]
			return totread, nil
		}
		copy(b, data)
		totread += len(data)
		b = b[len(data):]
		if len(b) == 0 {
			return totread, nil
		}
	}
	return totread, io.EOF
}

type randGen struct {
	src rand.Source
}

func NewFastRand() io.Reader {
	return &randGen{rand.NewSource(time.Now().UnixNano())}
}

func (r *randGen) Read(p []byte) (n int, err error) {
	todo := len(p)
	offset := 0
	for {
		val := int64(r.src.Int63())
		for i := 0; i < 8; i++ {
			p[offset] = byte(val & 0xff)
			todo--
			if todo == 0 {
				return len(p), nil
			}
			offset++
			val >>= 8
		}
	}

	panic("unreachable")
}
