package backend

import (
	"crypto/sha256"
	"errors"
	"io"
)

const (
	MinPrefixLength = 8
)

var (
	ErrNoIDPrefixFound   = errors.New("no ID found")
	ErrMultipleIDMatches = errors.New("multiple IDs with prefix found")
)

var (
	hashData = sha256.Sum256
)

const hashSize = sha256.Size

// Hash returns the ID for data.
func Hash(data []byte) ID {
	h := hashData(data)
	id := make([]byte, IDSize)
	copy(id, h[:])
	return id
}

// Find loads the list of all blobs of type t and searches for names which
// start with prefix. If none is found, nil and ErrNoIDPrefixFound is returned.
// If more than one is found, nil and ErrMultipleIDMatches is returned.
func Find(be Lister, t Type, prefix string) (string, error) {
	done := make(chan struct{})
	defer close(done)

	match := ""

	// TODO: optimize by sorting list etc.
	for name := range be.List(t, done) {
		if prefix == name[:len(prefix)] {
			if match == "" {
				match = name
			} else {
				return "", ErrMultipleIDMatches
			}
		}
	}

	if match != "" {
		return match, nil
	}

	return "", ErrNoIDPrefixFound
}

// FindSnapshot takes a string and tries to find a snapshot whose ID matches
// the string as closely as possible.
func FindSnapshot(be Lister, s string) (string, error) {
	// find snapshot id with prefix
	name, err := Find(be, Snapshot, s)
	if err != nil {
		return "", err
	}

	return name, nil
}

// PrefixLength returns the number of bytes required so that all prefixes of
// all names of type t are unique.
func PrefixLength(be Lister, t Type) (int, error) {
	done := make(chan struct{})
	defer close(done)

	// load all IDs of the given type
	list := make([]string, 0, 100)
	for name := range be.List(t, done) {
		list = append(list, name)
	}

	// select prefixes of length l, test if the last one is the same as the current one
outer:
	for l := MinPrefixLength; l < IDSize; l++ {
		var last string

		for _, name := range list {
			if last == name[:l] {
				continue outer
			}
			last = name[:l]
		}

		return l, nil
	}

	return IDSize, nil
}

// wrap around io.LimitedReader that implements io.ReadCloser
type blobReader struct {
	f      io.Closer
	rd     io.Reader
	closed bool
}

func (l *blobReader) Read(p []byte) (int, error) {
	n, err := l.rd.Read(p)
	if err == io.EOF {
		l.Close()
	}

	return n, err
}

func (l *blobReader) Close() error {
	if l == nil {
		return nil
	}

	if !l.closed {
		err := l.f.Close()
		l.closed = true
		return err
	}

	return nil
}

func LimitReader(f io.ReadCloser, n int64) *blobReader {
	return &blobReader{f: f, rd: io.LimitReader(f, n)}
}
