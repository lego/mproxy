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
	QueryFlagTailable
	QueryFlagSlaveOk
	QueryFlagLogReplay
	QueryFlagNoCursorTimeout
	QueryFlagAwaitData
)

func (f QueryOpFlags) String() string {
	var buf bytes.Buffer
	var flags []string
	if (f & QueryFlagTailable) != 0 {
		flags = append(flags, "tailable")
	}
	if (f & QueryFlagSlaveOk) != 0 {
		flags = append(flags, "slaveOk")
	}
	if (f & QueryFlagLogReplay) != 0 {
		flags = append(flags, "logReplay")
	}
	if (f & QueryFlagNoCursorTimeout) != 0 {
		flags = append(flags, "noCursorTimeout")
	}
	if (f & QueryFlagAwaitData) != 0 {
		flags = append(flags, "awaitData")
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
		return errors.Wrap(err, "failed to read Flags")
	}

	op.Collection = bytesutil.ReadCString(buf)

	if err := binary.Read(buf, binary.LittleEndian, &op.Skip); err != nil {
		return errors.Wrap(err, "failed to read Skip")
	}

	if err := binary.Read(buf, binary.LittleEndian, &op.Limit); err != nil {
		return errors.Wrap(err, "failed to read Limit")
	}

	bsonValue, err := bytesutil.ReadBSON(buf)
	if err != nil {
		return errors.Wrap(err, "failed to read Query")
	}
	op.Query = bsonValue

	if buf.Len() == 0 {
		return nil
	}

	bsonValue, err = bytesutil.ReadBSON(buf)
	if err != nil {
		return errors.Wrap(err, "failed to read Selector")
	}
	op.Selector = bsonValue

	return nil
}

func (op QueryOp) String() string {
	return fmt.Sprintf("<QueryOp Collection=%s Skip=%d Limit=%d Query=%v Selector=%v Flags=%s>", op.Collection, op.Skip, op.Limit, op.Query, op.Selector, op.Flags)
}

// &mgo.queryOp{collection:"test.$cmd", query:bson.D{bson.DocElem{Name:"insert", Value:"people"}, bson.DocElem{Name:"documents", Value:[]interface {}{(*main.Person)(0xc4200164b0)}}, bson.DocElem{Name:"writeConcern", Value:(*mgo.getLastError)(0xc4200188a0)}, bson.DocElem{Name:"ordered", Value:true}}, skip:0, limit:-1, selector:interface {}(nil), flags:0x0, replyFunc:(mgo.replyFunc)(0x1153180), mode:1, options:mgo.queryWrapper{Query:interface {}(nil), OrderBy:interface {}(nil), Hint:interface {}(nil), Explain:false, Snapshot:false, ReadPreference:bson.D(nil), MaxScan:0, MaxTimeMS:0, Comment:""}, hasOptions:false, serverTags:[]bson.D(nil)}
