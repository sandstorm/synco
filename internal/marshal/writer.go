package marshal

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
)

type Writer struct {
	io.Writer
	bw  *bufio.Writer
	enc *json.Encoder
}

func NewWriter(w io.Writer) *Writer {
	bw := bufio.NewWriter(w)

	return &Writer{
		Writer: w,
		bw:     bw,
		enc:    json.NewEncoder(bw),
	}
}

func (d *Writer) Close() error {
	return d.bw.Flush()
}

func (d *Writer) writePrefixed(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	err = binary.Write(d.bw, binary.LittleEndian, uint32(len(b)))
	if err != nil {
		return err
	}

	_, err = d.bw.Write(b)
	return err
}

func (d *Writer) WriteFileHeader(h *FileHeader) error {
	d.bw.WriteString("DUMP")

	return d.writePrefixed(h)
}

func (d *Writer) WriteTableHeader(h *TableHeader) error {
	d.bw.WriteByte(MarkerTable)

	return d.writePrefixed(h)
}

func (d *Writer) WriteRowData(r RowData) error {
	d.bw.WriteByte(MarkerRow)

	return d.writePrefixed(r)
}
