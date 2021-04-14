package marshal

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var ErrInvalidMarker = errors.New("invalid marker")

type Reader struct {
	r  io.Reader
	br *bufio.Reader
}

func NewReader(r io.Reader) *Reader {
	br := bufio.NewReader(r)

	return &Reader{
		r:  r,
		br: br,
	}
}

func (r *Reader) readLength() (len uint32, err error) {
	err = binary.Read(r.br, binary.LittleEndian, &len)
	return
}

func (r *Reader) decodePrefixed(v interface{}) error {
	len, err := r.readLength()
	if err != nil {
		return fmt.Errorf("read length: %w", err)
	}

	lr := io.LimitReader(r.br, int64(len))
	dec := json.NewDecoder(lr)

	return dec.Decode(v)
}

func (r *Reader) ReadFileHeader() (h *FileHeader, err error) {
	magic := make([]byte, 4)

	r.r.Read(magic)
	if string(magic) != "DUMP" {
		return nil, errors.New("invalid magic file string")
	}

	err = r.decodePrefixed(&h)
	return
}

func (r *Reader) ReadTableHeader() (h *TableHeader, err error) {
	m, err := r.br.ReadByte()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, err
		}

		return nil, fmt.Errorf("read marker: %w", err)
	}
	if m != MarkerTable {
		r.br.UnreadByte()
		return nil, ErrInvalidMarker
	}

	err = r.decodePrefixed(&h)
	return
}

func (r *Reader) ReadRows() (rows <-chan RowData, err <-chan error) {
	crows := make(chan []*string)
	cerr := make(chan error)

	go func() {
		defer close(crows)
		defer close(cerr)

		for {
			var d RowData

			m, err := r.br.ReadByte()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}

				cerr <- fmt.Errorf("read marker: %w", err)
				return
			}
			if m != MarkerRow {
				r.br.UnreadByte()
				break
			}

			err = r.decodePrefixed(&d)
			if err != nil {
				cerr <- err
				return
			}

			crows <- d
		}
	}()

	return crows, cerr
}

func (r *Reader) SkipRows() error {
	for {
		m, err := r.br.ReadByte()
		if err != nil {
			return fmt.Errorf("read row marker: %w", err)
		}
		if m != MarkerRow {
			r.br.UnreadByte()
			break
		}

		len, err := r.readLength()
		if err != nil {
			return fmt.Errorf("read row length: %w", err)
		}

		_, err = r.br.Discard(int(len))
		if err != nil {
			return fmt.Errorf("discard bytes: %w", err)
		}
	}

	return nil
}
