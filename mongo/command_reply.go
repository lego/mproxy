package mongo

import (
	"bytes"
	"fmt"

	"github.com/lego/mongotunnel/util/bytesutil"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2/bson"
)

type CommandReplyOp struct {
	Metadata    bson.D
	CommandReply bson.D
	OutputDocs   bson.D
}

func (op *CommandReplyOp) ReadFromBuffer(buf *bytes.Buffer) error {
	bsonValue, err := bytesutil.ReadBSON(buf)
	if err != nil {
		return errors.Wrap(err, "Failed to read Metadata")
	}
	op.Metadata = bsonValue

	bsonValue, err = bytesutil.ReadBSON(buf)
	if err != nil {
		return errors.Wrap(err, "Failed to read CommandReply")
	}
	op.CommandReply = bsonValue

	if buf.Len() != 0 {
	bsonValue, err = bytesutil.ReadBSON(buf)
	if err != nil {
		return errors.Wrap(err, "Failed to read OutputDocs")
	}
	op.OutputDocs = bsonValue
	}

	return nil
}

func (op CommandReplyOp) String() string {
	return fmt.Sprintf("<CommandReplyOp Metadata=%v CommandReply=%v OutputDocs=%v", op.Metadata, op.CommandReply, op.OutputDocs)
}
