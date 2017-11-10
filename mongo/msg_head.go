package mongo

import (
	"encoding/binary"
	"fmt"
	"io"
)

type MsgHead struct {
	TotalLen   int32
	ResponseID int32
	ResponseTo int32
	Opcode     Opcode
}

func NewMsgHead(op Op, responseID, responseTo int32) *MsgHead {
	return &MsgHead{
		TotalLen:   MsgHeadSize() + op.Size(),
		Opcode:     op.Opcode(),
		ResponseID: responseID,
		ResponseTo: responseTo,
	}
}

func (m *MsgHead) ReadFromBuffer(buf io.Reader) error {
	return binary.Read(buf, binary.LittleEndian, m)
}

func (m *MsgHead) WriteToBuffer(buf io.Writer) error {
	return binary.Write(buf, binary.LittleEndian, m)
}

func (m MsgHead) String() string {
	return fmt.Sprintf("<MsgHead Opcode=%s ResponseID=%d ReponseTo=%d TotalLen=%d>", m.Opcode, m.ResponseID, m.ResponseTo, m.TotalLen)
}

func MsgHeadSize() int32 {
	// 4 int32s
	return 4 * 4
}
