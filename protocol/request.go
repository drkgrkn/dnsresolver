package protocol

import (
	"encoding/binary"
	"encoding/hex"
	"io"
	"strings"
)

func UInt16ToByteSlice(u uint16) []byte {
	arr := make([]byte, 2)
	binary.BigEndian.PutUint16(arr[0:2], u)
	return arr
}
func UInt32ToByteSlice(u uint32) []byte {
	arr := make([]byte, 4)
	binary.BigEndian.PutUint32(arr[0:4], u)
	return arr
}

type Flags uint16

type MessageOptsFunc func(*Message)

func WithID(id uint16) func(*Message) {
	return func(r *Message) {
		r.Header.ID = id
	}
}
func WithRecursionDesired() func(*Message) {
	return func(r *Message) {
		mask := uint16(0x0100)
		r.Header.Flags |= mask
	}
}

func WithQuestion(name string, qType uint16, qClass uint16) func(*Message) {
	return func(r *Message) {
		q := Question{
			QName:  []byte(EncodeHostName(name)),
			QType:  qType,
			QClass: qClass,
		}
		r.Questions = append(r.Questions, q)
	}
}

type Header struct {
	ID      uint16
	Flags   uint16
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

type Question struct {
	QName  []byte
	QType  uint16
	QClass uint16
}

type Answer struct {
	Name     []byte
	Type     uint16
	Class    uint16
	TTL      uint32
	RDLength uint16
	RData    []byte
}

type Message struct {
	Header    Header
	Questions []Question
	Answers   []Answer
}

func NewRequest(opts ...MessageOptsFunc) Message {
	r := Message{
		Header: Header{
			ID:      0,
			Flags:   0,
			QDCount: 1,
			ANCount: 0,
			NSCount: 0,
			ARCount: 0,
		},
		Questions: []Question{},
		Answers:   []Answer{},
	}

	for _, f := range opts {
		f(&r)
	}

	return r
}

func (m Message) Encode() []byte {
	dst := make([]byte, 0)
	dst = hex.AppendEncode(dst, UInt16ToByteSlice(m.Header.ID))
	dst = hex.AppendEncode(dst, UInt16ToByteSlice(uint16(m.Header.Flags)))
	dst = hex.AppendEncode(dst, UInt16ToByteSlice(m.Header.QDCount))
	dst = hex.AppendEncode(dst, UInt16ToByteSlice(m.Header.ANCount))
	dst = hex.AppendEncode(dst, UInt16ToByteSlice(m.Header.NSCount))
	dst = hex.AppendEncode(dst, UInt16ToByteSlice(m.Header.ARCount))
	for _, q := range m.Questions {
		dst = hex.AppendEncode(dst, q.QName)
		dst = hex.AppendEncode(dst, UInt16ToByteSlice(q.QType))
		dst = hex.AppendEncode(dst, UInt16ToByteSlice(q.QClass))
	}
	for _, a := range m.Answers {
		dst = hex.AppendEncode(dst, a.Name)
		dst = hex.AppendEncode(dst, UInt16ToByteSlice(a.Type))
		dst = hex.AppendEncode(dst, UInt16ToByteSlice(a.Class))
		dst = hex.AppendEncode(dst, UInt32ToByteSlice(a.TTL))
		dst = hex.AppendEncode(dst, UInt16ToByteSlice(a.RDLength))
		dst = hex.AppendEncode(dst, a.RData)
	}
	return dst
}

func Decode(r io.Reader) Message {
	return Message{}
}

func EncodeString(s string) []byte {
	dst := make([]byte, 0)
	dst = append(dst, byte(len(s)))
	dst = append(dst, []byte(s)...)
	return dst
}

func EncodeHostName(hostName string) []byte {
	dst := make([]byte, 0)
	words := strings.Split(hostName, ".")

	for _, word := range words {
		dst = append(dst, byte(len(word)))
		dst = append(dst, []byte(word)...)
	}
	dst = append(dst, 0)

	return dst
}
