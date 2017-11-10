package proxy

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"

	"github.com/lego/mproxy/mongo"
	"github.com/lego/mproxy/util/context"
	"github.com/lego/mproxy/util/log"
)

// Proxy - Manages a Proxy connection, piping data between local and remote.
type Proxy struct {
	sentBytes     uint64
	receivedBytes uint64
	laddr, raddr  *net.TCPAddr
	lconn, rconn  io.ReadWriteCloser
	erred         bool
	errsig        chan bool
	tlsUnwrapp    bool
	tlsAddress    string

	Matcher  func([]byte)
	Replacer func([]byte) []byte

	// Settings
	Nagles    bool
	OutputHex bool
	ctx       *context.Context
}

// New - Create a new Proxy instance. Takes over local connection passed in,
// and closes it when finished.
func New(lconn *net.TCPConn, laddr, raddr *net.TCPAddr) *Proxy {
	return &Proxy{
		lconn:  lconn,
		laddr:  laddr,
		raddr:  raddr,
		erred:  false,
		errsig: make(chan bool),
		ctx:    context.NewContext(&log.NullLogger{}),
	}
}

// NewTLSUnwrapped - Create a new Proxy instance with a remote TLS server for
// which we want to unwrap the TLS to be able to connect without encryption
// locally
func NewTLSUnwrapped(lconn *net.TCPConn, laddr, raddr *net.TCPAddr, addr string) *Proxy {
	p := New(lconn, laddr, raddr)
	p.tlsUnwrapp = true
	p.tlsAddress = addr
	return p
}

type setNoDelayer interface {
	SetNoDelay(bool) error
}

func (p *Proxy) Ctx() *context.Context {
	return p.ctx
}

// Start - open connection to remote and start proxying data.
func (p *Proxy) Start() {
	defer p.lconn.Close()

	var err error
	//connect to remote
	if p.tlsUnwrapp {
		p.rconn, err = tls.Dial("tcp", p.tlsAddress, nil)
	} else {
		p.rconn, err = net.DialTCP("tcp", nil, p.raddr)
	}
	if err != nil {
		p.ctx.Log.Warn("Remote connection failed: %+v", err)
		return
	}
	defer p.rconn.Close()

	//nagles?
	if p.Nagles {
		if conn, ok := p.lconn.(setNoDelayer); ok {
			conn.SetNoDelay(true)
		}
		if conn, ok := p.rconn.(setNoDelayer); ok {
			conn.SetNoDelay(true)
		}
	}

	//display both ends
	p.ctx.Log.Info("Opened %s >>> %s", p.laddr.String(), p.raddr.String())

	//bidirectional copy
	go p.pipe(p.lconn, p.rconn)
	go p.pipe(p.rconn, p.lconn)

	//wait for close...

	<-p.errsig
	p.ctx.Log.Info("Closed (%d bytes sent, %d bytes recieved)", p.sentBytes, p.receivedBytes)
}

func (p *Proxy) err(s string, err error) {
	if p.erred {
		return
	}
	if err != io.EOF {
		p.ctx.Log.Warn(s, err)
	}
	p.errsig <- true
	p.erred = true
}

// func WriteInt(b *bytes.Buffer, i int32) {
// 	binary.Write(b, binary.LittleEndian, []byte{
// 		byte(i),
// 		byte(i >> 8),
// 		byte(i >> 16),
// 		byte(i >> 24),
// 	})
// }

func (p *Proxy) pipe(src, dst io.ReadWriter) {
	islocal := src == p.lconn

	var dataDirection string
	if islocal {
		dataDirection = ">>> %d bytes sent%s"
	} else {
		dataDirection = "<<< %d bytes recieved%s"
	}

	var byteFormat string
	if p.OutputHex {
		byteFormat = "%x"
	} else {
		byteFormat = "%s"
	}

	responseID := int32(400)

	//directional copy (64k buffer)
	buff := make([]byte, 0xffff)
	for {
		n, err := src.Read(buff)
		if err != nil {
			p.err("Read failed '%s'\n", err)
			return
		}
		b := buff[:n]
		if islocal {
			p.ctx.Log.LogC(log.Info, log.RedEmphasized, "INCOMING")
		} else {
			p.ctx.Log.LogC(log.Info, log.BlueEmphasized, "OUTGOING")
		}

		// //execute match
		// if p.Matcher != nil {
		// 	p.Matcher(b)
		// }
		//
		// //execute replace
		// if p.Replacer != nil {
		// 	b = p.Replacer(b)
		// }

		// out = mongo.AuthCmd{}
		// err = bson.Unmarshal(b, &out)
		// if err == nil {
		// 	p.ctx.Log.Debug("==== UNMARSHALLED ====%v\n", out)
		// 	p.ctx.Log.Debug("AuthCmd: %v\n", out)
		// } else {
		// 	p.ctx.Log.Debug("AuthCmd unmarshal failed '%s'\n", err)
		// }
		//
		mongobuf := bytes.NewBuffer(b)

		msgHead := mongo.MsgHead{}
		if err := msgHead.ReadFromBuffer(mongobuf); err != nil {
			p.ctx.Log.Warn("MsgHeader read failed: '%s'\n", err)
		}
		p.ctx.Log.Debug("   %s", msgHead)

		shouldDualWrite := false
		var dualWriteBytes []byte
		switch msgHead.Opcode {
		case mongo.Opcode_QUERY:
			queryOp := mongo.QueryOp{}
			if err := queryOp.ReadFromBuffer(mongobuf); err != nil {
				p.ctx.Log.Warn("failed to read queryOp: %+v", err)
			} else {
				p.ctx.Log.Debug("   %s", queryOp)
			}

			if isNegotiation(p.ctx, queryOp) {
				replyOp, err := createNegotiationReply(p.ctx, queryOp)
				if err != nil {
					p.ctx.Log.Warn("failed to create negotiation reply: %+v", err)
					panic("oops")
				}
				p.ctx.Log.LogC(log.Info, log.BlueEmphasized, "GENERATED OUTGOING")
				p.ctx.Log.Debug("   %s", replyOp)
				replyMsgHead := mongo.NewMsgHead(replyOp, 0 /* responseID */, 0 /* reponseTo */)
				replyMsgHead.WriteToBuffer(src)
				replyOp.WriteToBuffer(src)
				p.receivedBytes += uint64(replyMsgHead.TotalLen)
				continue
			} else if isQuery(p.ctx, queryOp) {
				p.ctx.Log.Debug("is a query!")
				replyOp, err := handleQuery(p.ctx, queryOp)
				if err != nil {
					p.ctx.Log.Warn("failed to handle query reply: %+v", err)
				} else {
					p.ctx.Log.LogC(log.Info, log.BlueEmphasized, "GENERATED OUTGOING")
					p.ctx.Log.Debug("   %s", replyOp)
					replyMsgHead := mongo.NewMsgHead(replyOp, responseID, msgHead.ResponseID)
					replyMsgHead.WriteToBuffer(src)
					replyOp.WriteToBuffer(src)
					p.receivedBytes += uint64(replyMsgHead.TotalLen)
					responseID++
					continue
				}
			}

			// if query.Map()["insert"] != nil {
			// 	p.ctx.Log.Debug("Inserting into: %s", query.Map()["insert"])

			// 	dualWriteTable := "employees"
			// 	query[0].Value = dualWriteTable
			// 	dualWriteQuery, err := bson.Marshal(query)
			// 	if err != nil {
			// 		p.ctx.Log.Debug("failed to unmarshal query: %+v", err)
			// 	}
			// 	p.ctx.Log.Debug("Dual write query: %v", query)

			// 	newBuf := bytes.NewBuffer([]byte{})
			// 	WriteInt(newBuf, totalLen)       // Modified after
			// 	WriteInt(newBuf, responseId+100) // Modified for uniqueness
			// 	WriteInt(newBuf, responseTo)
			// 	WriteInt(newBuf, opCode)
			// 	WriteInt(newBuf, int32(queryOp.Flags))
			// 	newBuf.WriteString(queryOp.Collection)
			// 	newBuf.WriteByte(0x0) // String null terminator
			// 	WriteInt(newBuf, queryOp.Skip)
			// 	WriteInt(newBuf, queryOp.Limit)
			// 	newBuf.Write(dualWriteQuery)
			// 	dualWriteBytesLocal := newBuf.Bytes()
			// 	newQueryLen := len(dualWriteBytesLocal)
			// 	dualWriteBytes = make([]byte, newQueryLen)
			// 	for i := 1; i < newQueryLen; i++ {
			// 		dualWriteBytes[i] = dualWriteBytesLocal[i]
			// 	}
			// 	shouldDualWrite = true
			// }
		case mongo.Opcode_COMMAND:
			commandOp := mongo.CommandOp{}
			if err := commandOp.ReadFromBuffer(mongobuf); err != nil {
				p.ctx.Log.Warn("failed to read commandOp: %+v", err)
			} else {
				p.ctx.Log.Debug("   commandOp=%s", commandOp)
			}
		case mongo.Opcode_COMMANDREPLY:
			commandReplyOp := mongo.CommandReplyOp{}
			if err := commandReplyOp.ReadFromBuffer(mongobuf); err != nil {
				p.ctx.Log.Warn("failed to read commandReplyOp: %+v", err)
			} else {
				p.ctx.Log.Debug("   commandReplyOp=%s", commandReplyOp)
			}
		case mongo.Opcode_REPLY:
			replyOp := mongo.ReplyOp{}
			if err := replyOp.ReadFromBuffer(mongobuf); err != nil {
				p.ctx.Log.Warn("failed to read replyOp: %+v", err)
			} else {
				p.ctx.Log.Debug("   %s", replyOp)
			}

			p.ctx.Log.Debug("documents: %#v", replyOp.Documents)

			// p.ctx.Log.Warn("negotiation replyOp docLen=%d", len(replyOp.Documents))
			// for i, doc := range replyOp.Documents {
			// 	p.ctx.Log.Warn("replyOp[%d]=%v", i, doc)
			// }
			// p.ctx.Log.Warn("neg replyMap=%v", replyOp.Documents.Map())
		default:
			panic(fmt.Sprintf("unhandled opcode=%s", msgHead.Opcode))
		}

		p.ctx.Log.Debug(dataDirection, n, "")
		p.ctx.Log.Trace(byteFormat, b)

		//write out result
		n, err = dst.Write(b)
		if err != nil {
			p.err("Write failed '%s'\n", err)
			return
		}
		if islocal {
			p.sentBytes += uint64(n)
		} else {
			p.receivedBytes += uint64(n)
		}

		// Dual write to separate table
		if shouldDualWrite {
			p.ctx.Log.Debug("=== DUAL WRITE BEGIN ===")
			p.ctx.Log.Debug(dataDirection, len(dualWriteBytes), "")
			p.ctx.Log.Trace(byteFormat, dualWriteBytes)
			p.ctx.Log.Debug("=== DUAL WRITE END ===")
			n, err = dst.Write(dualWriteBytes)
			if err == nil {
				p.ctx.Log.Debug("Dual write success")
			}
		}
	}
}
