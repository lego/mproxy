package mongo

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"gopkg.in/mgo.v2/bson"

	"github.com/lego/mproxy/util/bytesutil"
	"github.com/pkg/errors"
)

type QueryOpFlags uint32

const (
	_ QueryOpFlags = 1 << iota
	flagTailable
	flagSlaveOk
	flagLogReplay
	flagNoCursorTimeout
	flagAwaitData
)

type Mode int

const (
	// Relevant documentation on read preference modes:
	//
	//     http://docs.mongodb.org/manual/reference/read-preference/
	//
	Primary            Mode = 2 // Default mode. All operations read from the current replica set primary.
	PrimaryPreferred   Mode = 3 // Read from the primary if available. Read from the secondary otherwise.
	Secondary          Mode = 4 // Read from one of the nearest secondary members of the replica set.
	SecondaryPreferred Mode = 5 // Read from one of the nearest secondaries if available. Read from primary otherwise.
	Nearest            Mode = 6 // Read from one of the nearest members, irrespective of it being primary or secondary.

	// Read preference modes are specific to mgo:
	Eventual  Mode = 0 // Same as Nearest, but may change servers between reads.
	Monotonic Mode = 1 // Same as SecondaryPreferred before first write. Same as Primary after first write.
	Strong    Mode = 2 // Same as Primary.
)

type QueryOp struct {
	Flags      QueryOpFlags
	Collection string
	Skip       int32
	Limit      int32
	Query      bson.D
	Selector   bson.D
}

func (op *QueryOp) ReadFromBuffer(buf *bytes.Buffer) error {
	if err := binary.Read(buf, binary.LittleEndian, &op.Flags); err != nil {
		return errors.Wrap(err, "Failed to read Flags")
	}

	op.Collection = bytesutil.ReadCString(buf)

	if err := binary.Read(buf, binary.LittleEndian, &op.Skip); err != nil {
		return errors.Wrap(err, "Failed to read Skip")
	}

	if err := binary.Read(buf, binary.LittleEndian, &op.Limit); err != nil {
		return errors.Wrap(err, "Failed to read Limit")
	}

	bsonValue, err := bytesutil.ReadBSON(buf)
	if err != nil {
		return errors.Wrap(err, "Failed to read Query")
	}
	op.Query = bsonValue

	if buf.Len() == 0 {
		return nil
	}

	bsonValue, err = bytesutil.ReadBSON(buf)
	if err != nil {
		return errors.Wrap(err, "Failed to read Selector")
	}
	op.Selector = bsonValue

	return nil
}

func (op QueryOp) String() string {
	return fmt.Sprintf("<QueryOp Flags=%d Collection=%s Skip=%d Limit=%d Query=%v Selector=%v>", op.Flags, op.Collection, op.Skip, op.Limit, op.Query, op.Selector)
}

// &mgo.queryOp{collection:"test.$cmd", query:bson.D{bson.DocElem{Name:"insert", Value:"people"}, bson.DocElem{Name:"documents", Value:[]interface {}{(*main.Person)(0xc4200164b0)}}, bson.DocElem{Name:"writeConcern", Value:(*mgo.getLastError)(0xc4200188a0)}, bson.DocElem{Name:"ordered", Value:true}}, skip:0, limit:-1, selector:interface {}(nil), flags:0x0, replyFunc:(mgo.replyFunc)(0x1153180), mode:1, options:mgo.queryWrapper{Query:interface {}(nil), OrderBy:interface {}(nil), Hint:interface {}(nil), Explain:false, Snapshot:false, ReadPreference:bson.D(nil), MaxScan:0, MaxTimeMS:0, Comment:""}, hasOptions:false, serverTags:[]bson.D(nil)}
