package marshal

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
)

var ErrInvalidMarker = errors.New("invalid marker")

type Reader struct {
	r   io.Reader
	br  *bufio.Reader
	dec *json.Decoder
}

func NewReader(r io.Reader) *Reader {
	br := bufio.NewReader(r)

	return &Reader{
		r:   r,
		br:  br,
		dec: json.NewDecoder(br),
	}
}

func (r *Reader) decodePrefixed(v interface{}) error {
	var len uint32
	if err := binary.Read(r.br, binary.LittleEndian, &len); err != nil {
		return err
	}

	// lr := io.LimitReader(r.br, int64(len))
	// dec := json.NewDecoder(lr)

	// return dec.Decode(v)

	b := make([]byte, len)
	_, err := io.ReadFull(r.br, b)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, v)
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
		return nil, err
	}
	if m != MarkerTable {
		r.br.UnreadByte()
		return nil, ErrInvalidMarker
	}

	err = r.decodePrefixed(&h)
	return
}

func (r *Reader) ReadRows(c chan<- RowData) error {
	defer close(c)

	for {
		var d RowData

		m, err := r.br.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return err
		}
		if m != MarkerRow {
			r.br.UnreadByte()
			break
		}

		err = r.decodePrefixed(&d)
		if err != nil {
			return err
		}

		c <- d
	}

	return nil
}
