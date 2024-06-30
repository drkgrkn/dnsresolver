package protocol

import (
	"encoding/binary"
)

const (
	// record type
	RecordTypeA  uint16 = 1
	RecordTypeNS uint16 = 2

	// record class
	RecordClassIN uint16 = 1
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

type Header struct {
	ID      uint16
	Flags   uint16
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

func (h Header) offset() int {
	return 2 + 2 + 2 + 2 + 2 + 2
}

type Question struct {
	QName  []byte
	QType  uint16
	QClass uint16
}

func (q Question) offset() int {
	return len(q.QName) + 2 + 2
}

type answerName struct {
	isOffset bool
	name     []byte
	offset   uint16
}

func (an answerName) len() int {
	if an.isOffset {
		return 2
	} else {
		return len(an.name)
	}
}

func (an answerName) Bytes() []byte {
	if an.isOffset {
		return UInt16ToByteSlice(an.offset)
	} else {
		return an.name
	}
}

type Answer struct {
	Name     answerName
	Type     uint16
	Class    uint16
	TTL      uint32
	RDLength uint16
	RData    []byte
}

func (a Answer) offset() int {
	return a.Name.len() + 2 + 2 + 4 + 2 + len(a.RData)
}

type Message struct {
	Header    Header
	Questions []Question
	Answers   []Answer
}

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
			QName:  []byte(encodeHostName(name)),
			QType:  qType,
			QClass: qClass,
		}
		r.Questions = append(r.Questions, q)
	}
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
	dst = append(dst, UInt16ToByteSlice(m.Header.ID)...)
	dst = append(dst, UInt16ToByteSlice(m.Header.Flags)...)
	dst = append(dst, UInt16ToByteSlice(m.Header.QDCount)...)
	dst = append(dst, UInt16ToByteSlice(m.Header.ANCount)...)
	dst = append(dst, UInt16ToByteSlice(m.Header.NSCount)...)
	dst = append(dst, UInt16ToByteSlice(m.Header.ARCount)...)
	for _, q := range m.Questions {
		dst = append(dst, q.QName...)
		dst = append(dst, UInt16ToByteSlice(q.QType)...)
		dst = append(dst, UInt16ToByteSlice(q.QClass)...)
	}
	for _, a := range m.Answers {
		dst = append(dst, a.Name.Bytes()...)
		dst = append(dst, UInt16ToByteSlice(a.Type)...)
		dst = append(dst, UInt16ToByteSlice(a.Class)...)
		dst = append(dst, UInt32ToByteSlice(a.TTL)...)
		dst = append(dst, UInt16ToByteSlice(a.RDLength)...)
		dst = append(dst, a.RData...)
	}
	return dst
}

func (m Message) domainAtOffset(offset int) string {
	cur := 0
	cur += m.Header.offset()
	for _, q := range m.Questions {
		if cur == offset {
			name, _ := decodeHostName(q.QName)
			return name
		}
		cur += q.offset()
	}
	for _, a := range m.Answers {
		if cur == offset {
			name, _ := decodeHostName(a.Name.name)
			return name
		}
		cur += a.offset()
	}
	return ""
}

func (m Message) ToMap() map[string][][]byte {
	mappings := make(map[string][][]byte)
	for _, ans := range m.Answers {
		if ans.Name.isOffset {
			ns := m.domainAtOffset(int(ans.Name.offset))
			mappings[ns] = append(mappings[ns], ans.RData)
		}
	}

	return mappings
}
