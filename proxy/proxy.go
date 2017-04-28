package proxy

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"io"
	"net"

	"github.com/xLegoz/mproxy/mongo"
	"gopkg.in/mgo.v2/bson"
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

func readInt32(r *bytes.Buffer) (int32, error) {
	var b = make([]byte, 4)
	_, err := r.Read(b)
	if err != nil {
		return 0, err
	}
	return (int32(b[0])) |
		(int32(b[1]) << 8) |
		(int32(b[2]) << 16) |
		(int32(b[3]) << 24), nil
}

func readString(r *bytes.Buffer) string {
	str, _ := r.ReadString(0x0)
	return str[:len(str)-1]
}

func readBSON(buf *bytes.Buffer) ([]byte, error) {
	if buf.Len() < 4 {
		return nil, io.ErrUnexpectedEOF
	}

	length := binary.LittleEndian.Uint32(buf.Bytes())
	if int(length) > buf.Len() {
		return nil, io.ErrUnexpectedEOF
	}

	doc := make([]byte, length)
	_, err := io.ReadFull(buf, doc)
	return doc, err
}

func fill(r *bytes.Buffer, b []byte) error {
	l := len(b)
	n, err := r.Read(b)
	for n != l && err == nil {
		var ni int
		ni, err = r.Read(b[n:])
		n += ni
	}
	return err
}

func WriteInt(b *bytes.Buffer, i int32) {
	b.Write([]byte{
		byte(i),
		byte(i >> 8),
		byte(i >> 16),
		byte(i >> 24),
	})
}

func setInt32(b []byte, pos int, i int32) {
	b[pos] = byte(i)
	b[pos+1] = byte(i >> 8)
	b[pos+2] = byte(i >> 16)
	b[pos+3] = byte(i >> 24)
}

func setString(b []byte, pos int, i int32) {
	b[pos] = byte(i)
	b[pos+1] = byte(i >> 8)
	b[pos+2] = byte(i >> 16)
	b[pos+3] = byte(i >> 24)
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

		totalLen, _ := readInt32(mongobuf)
		responseId, _ := readInt32(mongobuf)
		responseTo, _ := readInt32(mongobuf)
		opCode, _ := readInt32(mongobuf)

		// header := mongo.MsgHeader{}
		// err = binary.Read(mongobuf, binary.BigEndian, &header)
		if err != nil {
			p.Log.Debug("MsgHeader read failed: '%s'\n", err)
		} else {
			p.Log.Debug("totalLen: %d", totalLen)
			p.Log.Debug("responseId: %d", responseId)
			p.Log.Debug("responseTo: %d", responseTo)
			p.Log.Debug("opCode: %d", opCode)
		}

		var query bson.D
		shouldDualWrite := false
		var dualWriteBytes []byte
		if opCode == 2004 {
			var flags mongo.QueryOpFlags
			err = binary.Read(mongobuf, binary.LittleEndian, &flags)
			if err != nil {
				p.Log.Debug("Failed to read flags: %s", err)
			}

			var collection = readString(mongobuf)
			if err != nil {
				p.Log.Debug("Failed to read collection: %s", err)
			}

			p.Log.Debug("Collection: %s", collection)

			var skip int32

			err = binary.Read(mongobuf, binary.LittleEndian, &skip)
			if err != nil {
				p.Log.Debug("Failed to read skip: %s", err)
			}

			p.Log.Debug("Skip: %d", skip)

			var limit int32

			err = binary.Read(mongobuf, binary.LittleEndian, &limit)
			if err != nil {
				p.Log.Debug("Failed to read limit: %s", err)
			}

			p.Log.Debug("Limit: %d", limit)

			rawBson, err := readBSON(mongobuf)
			if err != nil {
				p.Log.Debug("Failed to parse query: %s", err)
			}

			err = bson.Unmarshal(rawBson, &query)
			if err != nil {
				p.Log.Debug("Failed to unmarshal query: %s", err)
			}
			p.Log.Debug("QUERY: %v", query)

			if query.Map()["insert"] != nil {
				p.Log.Debug("Inserting into: %s", query.Map()["insert"])

				dualWriteTable := "employees"
				query[0].Value = dualWriteTable
				dualWriteQuery, err := bson.Marshal(query)
				if err != nil {
					p.Log.Debug("Failed to unmarshal query: %s", err)
				}
				p.Log.Debug("Dual write query: %v", query)

				newBuf := bytes.NewBuffer([]byte{})
				WriteInt(newBuf, totalLen)       // Modified after
				WriteInt(newBuf, responseId+100) // Modified for uniqueness
				WriteInt(newBuf, responseTo)
				WriteInt(newBuf, opCode)
				WriteInt(newBuf, int32(flags))
				newBuf.WriteString(collection)
				newBuf.WriteByte(0x0) // String null terminator
				WriteInt(newBuf, skip)
				WriteInt(newBuf, limit)
				newBuf.Write(dualWriteQuery)
				dualWriteBytesLocal := newBuf.Bytes()
				newQueryLen := len(dualWriteBytesLocal)
				dualWriteBytes = make([]byte, newQueryLen)
				for i := 1; i < newQueryLen; i++ {
					dualWriteBytes[i] = dualWriteBytesLocal[i]
				}
				shouldDualWrite = true
			}
		}

		// 	err = bson.Unmarshal(queryop_bytes, &out)
		// 	if err == nil {
		// 		p.Log.Debug("==== UNMARSHALLED ====")
		// 		p.Log.Debug("QueryOp: %v", out)
		// 	} else {
		// 		p.Log.Debug("QueryOp unmarshal failed '%s'", err)
		// 		p.Log.Debug("Bytes: %b", queryop_bytes)
		//
		// 	}
		// } else {
		// }
		//show output
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
