package mongo

import (
	"bytes"
	"fmt"

	"github.com/lego/mongotunnel/util/bytesutil"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2/bson"
)

type CommandOp struct {
	Database    string
	Command     string
	Metadata    bson.D
	CommandArgs bson.D
	InputDocs   bson.D
}

func (op *CommandOp) ReadFromBuffer(buf *bytes.Buffer) error {
	op.Database = bytesutil.ReadCString(buf)
	op.Command = bytesutil.ReadCString(buf)

	bsonValue, err := bytesutil.ReadBSON(buf)
	if err != nil {
		return errors.Wrap(err, "Failed to read Metadata")
	}
	op.Metadata = bsonValue

	bsonValue, err = bytesutil.ReadBSON(buf)
	if err != nil {
		return errors.Wrap(err, "Failed to read CommandArgs")
	}
	op.CommandArgs = bsonValue

	if buf.Len() != 0 {
	bsonValue, err = bytesutil.ReadBSON(buf)
	if err != nil {
		return errors.Wrap(err, "Failed to read InputDocs")
	}
	op.InputDocs = bsonValue
	}

	return nil
}

func (op CommandOp) String() string {
	return fmt.Sprintf("<CommandOp Database=%q Command=%q Metadata=%v CommandArgs=%v InputDocs=%v", op.Database, op.Command, op.Metadata, op.CommandArgs, op.InputDocs)
}
