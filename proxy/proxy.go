package proxy

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"

	"github.com/lego/mproxy/mongo"
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
	Log       Logger
	OutputHex bool
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
		Log:    NullLogger{},
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
		p.Log.Warn("Remote connection failed: %s", err)
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
	p.Log.Info("Opened %s >>> %s", p.laddr.String(), p.raddr.String())

	//bidirectional copy
	go p.pipe(p.lconn, p.rconn)
	go p.pipe(p.rconn, p.lconn)

	//wait for close...
	<-p.errsig
	p.Log.Info("Closed (%d bytes sent, %d bytes recieved)", p.sentBytes, p.receivedBytes)
}

func (p *Proxy) err(s string, err error) {
	if p.erred {
		return
	}
	if err != io.EOF {
		p.Log.Warn(s, err)
	}
	p.errsig <- true
	p.erred = true
}

func WriteInt(b *bytes.Buffer, i int32) {
	// FIXME(joey): replace with binary.Write(..., binary.LittleEndian, ...)
	b.Write([]byte{
		byte(i),
		byte(i >> 8),
		byte(i >> 16),
		byte(i >> 24),
	})
}

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

	//directional copy (64k buffer)
	buff := make([]byte, 0xffff)
	for {
		n, err := src.Read(buff)
		if err != nil {
			p.err("Read failed '%s'\n", err)
			return
		}
		b := buff[:n]
		p.Log.Debug("=====NEW MESSAGE=====")

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
		// 	p.Log.Debug("==== UNMARSHALLED ====%v\n", out)
		// 	p.Log.Debug("AuthCmd: %v\n", out)
		// } else {
		// 	p.Log.Debug("AuthCmd unmarshal failed '%s'\n", err)
		// }
		//
		mongobuf := bytes.NewBuffer(b)

		msgHead := mongo.MsgHead{}
		if err := msgHead.ReadFromBuffer(mongobuf); err != nil {
			p.Log.Debug("MsgHeader read failed: '%s'\n", err)
		}
		p.Log.Debug("   msgHead=%s", msgHead)

		shouldDualWrite := false
		var dualWriteBytes []byte
		switch msgHead.Opcode {
		case mongo.Opcode_QUERY:
			queryOp := mongo.QueryOp{}
			if err := queryOp.ReadFromBuffer(mongobuf); err != nil {
				p.Log.Debug("Failed to read queryOp: %s", err)
			} else {
				p.Log.Debug("   queryOp=%s", queryOp)
			}

			// if query.Map()["insert"] != nil {
			// 	p.Log.Debug("Inserting into: %s", query.Map()["insert"])

			// 	dualWriteTable := "employees"
			// 	query[0].Value = dualWriteTable
			// 	dualWriteQuery, err := bson.Marshal(query)
			// 	if err != nil {
			// 		p.Log.Debug("Failed to unmarshal query: %s", err)
			// 	}
			// 	p.Log.Debug("Dual write query: %v", query)

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
				p.Log.Debug("Failed to read commandOp: %s", err)
			} else {
				p.Log.Debug("   commandOp=%s", commandOp)
			}
		case mongo.Opcode_COMMANDREPLY:
			commandReplyOp := mongo.CommandReplyOp{}
			if err := commandReplyOp.ReadFromBuffer(mongobuf); err != nil {
				p.Log.Debug("Failed to read commandReplyOp: %s", err)
			} else {
				p.Log.Debug("   commandReplyOp=%s", commandReplyOp)
			}
		case mongo.Opcode_REPLY:
			replyOp := mongo.ReplyOp{}
			if err := replyOp.ReadFromBuffer(mongobuf); err != nil {
				p.Log.Debug("Failed to read replyOp: %s", err)
			} else {
				p.Log.Debug("   replyOp=%s", replyOp)
			}
		default:
			p.Log.Warn("unhandled opcode=%d", msgHead.Opcode)
		}

		p.Log.Debug(dataDirection, n, "")
		p.Log.Trace(byteFormat, b)

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
			p.Log.Debug("=== DUAL WRITE BEGIN ===")
			p.Log.Debug(dataDirection, len(dualWriteBytes), "")
			p.Log.Trace(byteFormat, dualWriteBytes)
			p.Log.Debug("=== DUAL WRITE END ===")
			n, err = dst.Write(dualWriteBytes)
			if err == nil {
				p.Log.Debug("Dual write success")
			}
		}
	}
}
