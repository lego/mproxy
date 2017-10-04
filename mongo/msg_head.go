package mongo

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type MsgHead struct {
	TotalLen   int32
	ResponseID int32
	ResponseTo int32
	Opcode     Opcode
}

func (m *MsgHead) ReadFromBuffer(buf *bytes.Buffer) error {
	binary.Read(buf, binary.LittleEndian, m)
	return nil
}

func (m MsgHead) String() string {
	return fmt.Sprintf("<MsgHead TotalLen=%d ResponseID=%d ReponseTo=%d Opcode=%d>", m.TotalLen, m.ResponseID, m.ResponseTo, m.Opcode)
}
