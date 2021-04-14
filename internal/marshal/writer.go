package marshal

import (
	"encoding/binary"
	"encoding/json"
	"io"
)

type Writer struct {
	w io.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w: w,
	}
}

func (d *Writer) writePrefixed(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	err = binary.Write(d.w, binary.LittleEndian, uint32(len(b)))
	if err != nil {
		return err
	}

	_, err = d.w.Write(b)
	return err
}

func (d *Writer) WriteFileHeader(h *FileHeader) error {
	d.w.Write([]byte("DUMP"))

	return d.writePrefixed(h)
}

func (d *Writer) WriteTableHeader(h *TableHeader) error {
	d.w.Write([]byte{MarkerTable})

	return d.writePrefixed(h)
}

func (d *Writer) WriteRowData(r RowData) error {
	d.w.Write([]byte{MarkerRow})

	buf := make([]byte, binary.MaxVarintLen64)

	for _, v := range r {
		// Write null marker: 0 if null, 1 if not
		if v == nil {
			d.w.Write([]byte{0})
			continue
		} else {
			d.w.Write([]byte{1})
		}

		// Encode the value length as a varint
		n := binary.PutUvarint(buf, uint64(len(*v)))
		d.w.Write(buf[:n])

		// Write the string value as bytes
		d.w.Write([]byte(*v))
	}

	return nil
}
