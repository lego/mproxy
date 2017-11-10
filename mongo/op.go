package mongo

import (
	"fmt"
	"io"
)

type Opcode int32

const (
	// Opcode_REPLY
	// Reply to a client request. responseTo is set.
	Opcode_REPLY Opcode = 1
	// Opcode_UPDATE
	// Update document.
	Opcode_UPDATE Opcode = 2001
	// Opcode_INSERT
	// Insert new document.
	Opcode_INSERT Opcode = 2002
	// Opcode_RESERVED
	// Formerly used for OP_GET_BY_OID.
	Opcode_RESERVED Opcode = 2003
	// Opcode_QUERY
	// Query a collection.
	Opcode_QUERY Opcode = 2004
	// Opcode_GET_MORE
	// Get more data from a query. See Cursors.
	Opcode_GET_MORE Opcode = 20
	// Opcode_DELETE
	// Delete documents.
	Opcode_DELETE Opcode = 2006
	// Opcode_KILL_CURSORS
	// Notify database that the client has finished with the cursor.
	Opcode_KILL_CURSORS Opcode = 2007
	// Opcode_COMMAND
	// Cluster internal protocol representing a command request.
	Opcode_COMMAND Opcode = 2010
	// Opcode_COMMANDREPLY
	// Cluster internal protocol representing a reply to an OP_COMMAND.
	Opcode_COMMANDREPLY Opcode = 2011
)

func (o Opcode) String() string {
	switch o {
	case Opcode_REPLY:
		return "REPLY"
	case Opcode_UPDATE:
		return "UPDATE"
	case Opcode_INSERT:
		return "INSERT"
	case Opcode_RESERVED:
		return "RESERVED"
	case Opcode_QUERY:
		return "QUERY"
	case Opcode_GET_MORE:
		return "GET_MORE"
	case Opcode_DELETE:
		return "DELETE"
	case Opcode_KILL_CURSORS:
		return "KILL_CURSORS"
	case Opcode_COMMAND:
		return "COMMAND"
	case Opcode_COMMANDREPLY:
		return "COMMANDREPLY"
	default:
		panic(fmt.Sprintf("unhandled Opcode: %d", o))
	}
}

type Op interface {
	// FIXME(joey): Read/Write should be different. Maybe just implement
	// io.Writer / io.Reader interface.
	ReadFromBuffer(io.Reader) error
	WriteToBuffer(io.Writer) error
	Size() int32
	Opcode() Opcode
}

var _ Op = (*ReplyOp)(nil)
