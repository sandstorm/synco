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

	isSkipping bool
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

func (r *Reader) ReadRows(ncol int) (rows <-chan RowData, err <-chan error) {
	crows := make(chan []*string)
	cerr := make(chan error)

	go func() {
		defer close(crows)
		defer close(cerr)

		for {
			d := make([]*string, ncol)

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

			err = r.readRow(d)
			if err != nil {
				cerr <- fmt.Errorf("read row: %w", err)
				return
			}

			crows <- d
		}
	}()

	return crows, cerr
}

func (r *Reader) readRow(cols []*string) error {
	for i := 0; i < len(cols); i++ {
		// Read null marker
		nullMarker, err := r.br.ReadByte()
		if err != nil {
			return fmt.Errorf("read null marker: %w", err)
		}

		// If it's 0, the value is null and we continue with the next value
		if nullMarker == 0 {
			cols[i] = nil
			continue
		}

		// Peek enough bytes to decode a varint
		buf, err := r.br.Peek(binary.MaxVarintLen64)
		if err != nil {
			return fmt.Errorf("read data length: %w", err)
		}

		// Decode the varint from the peeked bytes
		len, n := binary.Uvarint(buf)
		if n <= 0 {
			return errors.New("failed to decode data length")
		}

		// Discard the bytes the varint used and advance
		r.br.Discard(n)

		if r.isSkipping {
			r.br.Discard(int(len))
		} else {
			buf = make([]byte, len)
			_, err = io.ReadFull(r.br, buf)
			if err != nil {
				return fmt.Errorf("read value: %w", err)
			}

			str := string(buf)
			cols[i] = &str
		}
	}

	return nil
}

func (r *Reader) SkipRows(ncol int) error {
	cols := make([]*string, ncol)

	r.isSkipping = true
	defer func() {
		r.isSkipping = false
	}()

	for {
		m, err := r.br.ReadByte()
		if err != nil {
			return fmt.Errorf("read row marker: %w", err)
		}
		if m != MarkerRow {
			r.br.UnreadByte()
			break
		}

		err = r.readRow(cols)
		if err != nil {
			return fmt.Errorf("skip row: %w", err)
		}
	}

	return nil
}
