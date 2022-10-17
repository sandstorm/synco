package marshal

import (
	"encoding/binary"
	"io"
)

func writeString(w io.Writer, s string) (err error) {
	if err = binary.Write(w, binary.LittleEndian, uint32(len(s))); err != nil {
		return
	}
	if _, err = w.Write([]byte(s)); err != nil {
		return
	}
	return
}

func readString(r io.Reader) (s string, err error) {
	var len uint32

	if err = binary.Read(r, binary.LittleEndian, &len); err != nil {
		return
	}

	b := make([]byte, len)
	if _, err = r.Read(b); err != nil {
		return
	}

	return string(b), nil
}
