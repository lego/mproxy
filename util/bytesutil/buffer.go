package bytesutil

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/pkg/errors"
	"gopkg.in/mgo.v2/bson"
)

func ReadCString(b *bytes.Buffer) string {
	str, _ := b.ReadString(0x0)
	return str[:len(str)-1]
}

func readRawBSON(buf io.Reader) ([]byte, error) {
	// FIXME(joey): Screw this interface.
	// if buf.Len() < 4 {
	// 	return nil, io.ErrUnexpectedEOF
	// }

	var length uint32
	if err := binary.Read(buf, binary.LittleEndian, &length); err != nil {
		return nil, errors.Wrap(err, "failed to read length")
	}
	// length := binary.LittleEndian.Uint32(buf.Bytes())
	// FIXME(joey): Screw this interface.
	// if int(length) > buf.Len() {
	// 	return nil, io.ErrUnexpectedEOF
	// }

	// Length includes the size of length, so remove 4 bytes.
	doc := make([]byte, length)
	binary.LittleEndian.PutUint32(doc, length)
	n, err := io.ReadFull(buf, doc[4:])
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read document bytes, read %d bytes", n)
	}
	return doc, nil
}

func ReadBSON(buf io.Reader) (bson.D, error) {
	docRawBSON, err := readRawBSON(buf)
	if err != nil {
		return nil, err
	}
	var doc bson.D
	if err := bson.Unmarshal(docRawBSON, &doc); err != nil {
		return nil, errors.Wrap(err, "failed to read bson.D")
	}
	return doc, nil
}

// WriteBSON
// Returns the bytes written.
func WriteBSON(buf io.Writer, d bson.D) (int, error) {
	out, err := bson.Marshal(d)
	if err != nil {
		return 0, errors.Wrap(err, "failed to marshal bson.D")
	}
	return buf.Write(out)
}
