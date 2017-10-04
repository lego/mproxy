package bytesutil

import (
	"bytes"
	"encoding/binary"
	"io"

	"gopkg.in/mgo.v2/bson"
)

func ReadCString(b *bytes.Buffer) string {
	str, _ := b.ReadString(0x0)
	return str[:len(str)-1]
}

func readRawBSON(buf *bytes.Buffer) ([]byte, error) {
	if buf.Len() < 4 {
		return nil, io.ErrUnexpectedEOF
	}

	length := binary.LittleEndian.Uint32(buf.Bytes())
	if int(length) > buf.Len() {
		return nil, io.ErrUnexpectedEOF
	}

	doc := make([]byte, length)
	_, err := io.ReadFull(buf, doc)
	return doc, err
}

func ReadBSON(buf *bytes.Buffer) (bson.D, error) {
	docRawBSON, err := readRawBSON(buf)
	if err != nil {
		return nil, err
	}
	var doc bson.D
	if err := bson.Unmarshal(docRawBSON, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}
