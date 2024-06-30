package protocol

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
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

func (h Header) WriteTo(w io.Writer) (int64, error) {
	sum := 0
	n, err := w.Write(UInt16ToByteSlice(h.ID))
	sum += n
	if err != nil {
		return int64(sum), err
	}
	n, err = w.Write(UInt16ToByteSlice(h.Flags))
	sum += n
	if err != nil {
		return int64(sum), err
	}
	n, err = w.Write(UInt16ToByteSlice(h.QDCount))
	sum += n
	if err != nil {
		return int64(sum), err
	}
	n, err = w.Write(UInt16ToByteSlice(h.ANCount))
	sum += n
	if err != nil {
		return int64(sum), err
	}
	n, err = w.Write(UInt16ToByteSlice(h.NSCount))
	sum += n
	if err != nil {
		return int64(sum), err
	}
	n, err = w.Write(UInt16ToByteSlice(h.ARCount))
	sum += n
	if err != nil {
		return int64(sum), err
	}

	return int64(sum), nil
}

type Question struct {
	QName  DomainName
	QType  uint16
	QClass uint16
}

func (q Question) offset() int {
	return q.QName.len() + 2 + 2
}

func (q Question) WriteTo(w io.Writer) (int64, error) {
	sum := 0
	n, err := w.Write(q.QName.Bytes())
	sum += n
	if err != nil {
		return int64(sum), err
	}
	n, err = w.Write(UInt16ToByteSlice(q.QType))
	sum += n
	if err != nil {
		return int64(sum), err
	}
	n, err = w.Write(UInt16ToByteSlice(q.QClass))
	sum += n
	if err != nil {
		return int64(sum), err
	}
	return int64(sum), nil
}

type Answer struct {
	Name     DomainName
	Type     uint16
	Class    uint16
	TTL      uint32
	RDLength uint16
	RData    []byte
}

func (a Answer) offset() int {
	return a.Name.len() + 2 + 2 + 4 + 2 + len(a.RData)
}

func (a Answer) Result() string {
	switch a.Type {
	case RecordTypeA:
		return fmt.Sprintf("%d.%d.%d.%d",
			a.RData[0],
			a.RData[1],
			a.RData[2],
			a.RData[3],
		)
	default:
		return ""
	}
}

func (a Answer) WriteTo(w io.Writer) (int64, error) {
	sum := 0
	n, err := w.Write(a.Name.Bytes())
	sum += n
	if err != nil {
		return int64(sum), err
	}
	n, err = w.Write(UInt16ToByteSlice(a.Type))
	sum += n
	if err != nil {
		return int64(sum), err
	}
	n, err = w.Write(UInt16ToByteSlice(a.Class))
	sum += n
	if err != nil {
		return int64(sum), err
	}
	n, err = w.Write(UInt32ToByteSlice(a.TTL))
	sum += n
	if err != nil {
		return int64(sum), err
	}
	n, err = w.Write(UInt16ToByteSlice(a.RDLength))
	sum += n
	if err != nil {
		return int64(sum), err
	}
	n, err = w.Write(a.RData)
	sum += n
	if err != nil {
		return int64(sum), err
	}
	return int64(sum), nil
}

type Message struct {
	Header    Header
	Questions []Question
	Answers   []Answer

	msg []byte
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
			QName:  newDomainName(name),
			QType:  qType,
			QClass: qClass,
		}
		r.Questions = append(r.Questions, q)
	}
}

func NewMessage(opts ...MessageOptsFunc) Message {
	msg := Message{
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

		msg: []byte{},
	}

	for _, f := range opts {
		f(&msg)
	}

	msg.msg = msg.encode()
	return msg
}

func (m Message) Bytes() []byte {
	return m.msg
}

func (m Message) encode() []byte {
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	m.Header.WriteTo(w)
	for _, q := range m.Questions {
		q.WriteTo(w)
	}
	for _, a := range m.Answers {
		a.WriteTo(w)
	}
	w.Flush()
	return b.Bytes()
}

func (m Message) labelAtOffset(i int) (DomainName, bool) {
	curr := m.Header.offset()
	for _, q := range m.Questions {
		if curr == i {
			return q.QName, true
		}
		curr += q.offset()
	}
	for _, a := range m.Answers {
		if curr == i {
			return a.Name, true
		}
		curr += a.offset()
	}

	return DomainName{}, false
}

func (m Message) fullNameOfAnswer(a Answer) string {
	var sb strings.Builder
	cur := a.Name
outer:
	for {
		for _, l := range cur.labels {
			if l.isZero() {
				break outer
			}
			if l.isOffset() {
				next, _ := m.labelAtOffset(l.offset())
				cur = next
				break
			}
			sb.WriteString(l.str)
			sb.WriteByte('.')
		}
	}
	result := sb.String()
	return result[:len(result)-1]
}

func (m Message) RecordsOfDomainName(dns string) []string {
	results := make([]string, 0)

	for _, a := range m.Answers {
		fullName := m.fullNameOfAnswer(a)
		if fullName == dns {
			results = append(results, a.Result())
		}
	}

	return results
}
