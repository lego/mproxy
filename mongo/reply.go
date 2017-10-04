package mongo

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"gopkg.in/mgo.v2/bson"

	"github.com/lego/mproxy/util/bytesutil"
	"github.com/pkg/errors"
)

type ReplyOp struct {
	Flags     uint32
	CursorID  int64
	FirstDoc  int32
	ReplyDocs int32
	Documents bson.D
}

func (op *ReplyOp) ReadFromBuffer(buf *bytes.Buffer) error {
	if err := binary.Read(buf, binary.LittleEndian, &op.Flags); err != nil {
		return errors.Wrap(err, "Failed to read Flags")
	}

	if err := binary.Read(buf, binary.LittleEndian, &op.CursorID); err != nil {
		return errors.Wrap(err, "Failed to read CursorID")
	}

	if err := binary.Read(buf, binary.LittleEndian, &op.FirstDoc); err != nil {
		return errors.Wrap(err, "Failed to read FirstDoc")
	}

	if err := binary.Read(buf, binary.LittleEndian, &op.ReplyDocs); err != nil {
		return errors.Wrap(err, "Failed to read ReplyDocs")
	}

	if buf.Len() == 0 {
		return nil
	}

	bsonValue, err := bytesutil.ReadBSON(buf)
	if err != nil {
		return errors.Wrap(err, "Failed to read Documents")
	}
	op.Documents = bsonValue

	return nil
}

func (op ReplyOp) String() string {
	return fmt.Sprintf("<ReplyOp Flags=%d CursorID=%d FirstDoc=%d ReplyDocs=%d Documents=%v>", op.Flags, op.CursorID, op.FirstDoc, op.ReplyDocs, op.Documents)
}
