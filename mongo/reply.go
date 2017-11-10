package mongo

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"gopkg.in/mgo.v2/bson"

	"github.com/lego/mproxy/util/bytesutil"
	"github.com/pkg/errors"
)

type ReplyOpFlags uint32

const (
	_ ReplyOpFlags = 1 << iota
	ReplyFlagCursorNotFound
	ReplyFlagQueryFailure
	ReplyFlagShardConfigStale
	ReplyFlagAwaitCapable
)

func (f ReplyOpFlags) String() string {
	var buf bytes.Buffer
	var flags []string
	if (f & ReplyFlagCursorNotFound) != 0 {
		flags = append(flags, "cursotNotFound")
	}
	if (f & ReplyFlagQueryFailure) != 0 {
		flags = append(flags, "queryFailure")
	}
	if (f & ReplyFlagShardConfigStale) != 0 {
		flags = append(flags, "shardConfigStale")
	}
	if (f & ReplyFlagAwaitCapable) != 0 {
		flags = append(flags, "awaitCapable")
	}

	buf.WriteByte('[')
	for i, flag := range flags {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(flag)
	}
	buf.WriteByte(']')
	return buf.String()
}

type ReplyOp struct {
	Flags     ReplyOpFlags
	CursorID  int64
	FirstDoc  int32
	ReplyDocs int32
	Documents bson.D
}

func (op *ReplyOp) ReadFromBuffer(buf io.Reader) error {
	if err := binary.Read(buf, binary.LittleEndian, &op.Flags); err != nil {
		return errors.Wrap(err, "failed to read Flags")
	}

	if err := binary.Read(buf, binary.LittleEndian, &op.CursorID); err != nil {
		return errors.Wrap(err, "failed to read CursorID")
	}

	if err := binary.Read(buf, binary.LittleEndian, &op.FirstDoc); err != nil {
		return errors.Wrap(err, "failed to read FirstDoc")
	}

	if err := binary.Read(buf, binary.LittleEndian, &op.ReplyDocs); err != nil {
		return errors.Wrap(err, "failed to read ReplyDocs")
	}

	// FIXME(joey): May need to handle something here...

	// if buf.Len() == 0 {
	// 	return nil
	// }

	bsonValue, err := bytesutil.ReadBSON(buf)
	if err != nil {
		return errors.Wrap(err, "failed to read Documents")
	}
	op.Documents = bsonValue

	return nil
}

func (op *ReplyOp) WriteToBuffer(buf io.Writer) error {
	if err := binary.Write(buf, binary.LittleEndian, &op.Flags); err != nil {
		return errors.Wrap(err, "failed to write Flags")
	}

	if err := binary.Write(buf, binary.LittleEndian, &op.CursorID); err != nil {
		return errors.Wrap(err, "failed to write CursorID")
	}

	if err := binary.Write(buf, binary.LittleEndian, &op.FirstDoc); err != nil {
		return errors.Wrap(err, "failed to write FirstDoc")
	}

	if err := binary.Write(buf, binary.LittleEndian, &op.ReplyDocs); err != nil {
		return errors.Wrap(err, "failed to write ReplyDocs")
	}

	if _, err := bytesutil.WriteBSON(buf, op.Documents); err != nil {
		return errors.Wrap(err, "failed to write Documents")
	}
	return nil
}

func (op *ReplyOp) Size() int32 {
	// FIXME(joey): This absolutely SUCKS. We can probably do better by
	// caching the marshalled bytes for later.
	bytes, err := bson.Marshal(op.Documents)
	if err != nil {
		panic(fmt.Sprintf("got error while trying to marshal ReplyOp.Documents: %+v", err))
	}
	// 3 int32, 1 int64, documents
	return 3*4 + 8 + int32(len(bytes))
}

func (op *ReplyOp) Opcode() Opcode {
	return Opcode_REPLY
}

func (op ReplyOp) String() string {
	return fmt.Sprintf("<ReplyOp CursorID=%d FirstDoc=%d ReplyDocs=%d Documents=%v Flags=%s>", op.CursorID, op.FirstDoc, op.ReplyDocs, op.Documents, op.Flags)
}
