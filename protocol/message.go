package protocol

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
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

type ResourceRecord struct {
	Name     DomainName
	Type     uint16
	Class    uint16
	TTL      uint32
	RDLength uint16
	RData    DomainName
}

func (a ResourceRecord) offsetWithoutRData() int {
	return a.Name.len() + 2 + 2 + 4 + 2
}

func (a ResourceRecord) offset() int {
	return a.Name.len() + 2 + 2 + 4 + 2 + int(a.RDLength)
}

func (m Message) formattedRDataOf(a ResourceRecord) string {
	switch a.Type {
	case RecordTypeA:
		return a.RData.labels[0].str
	case RecordTypeCNAME:
		return m.fullRDataOfRecord(a)
	case RecordTypeAAAA:
		return ""
	default:
		return ""
	}
}

func (a ResourceRecord) WriteTo(w io.Writer) (int64, error) {
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
	n, err = w.Write(a.RData.Bytes())
	sum += n
	if err != nil {
		return int64(sum), err
	}
	return int64(sum), nil
}

type Message struct {
	Header     Header
	Questions  []Question
	Answers    []ResourceRecord
	Authority  []ResourceRecord
	Additional []ResourceRecord

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
		Header:     Header{ID: 0, Flags: 0, QDCount: 1, ANCount: 0, NSCount: 0, ARCount: 0},
		Questions:  []Question{},
		Answers:    []ResourceRecord{},
		Authority:  []ResourceRecord{},
		Additional: []ResourceRecord{},
		msg:        []byte{},
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
	for _, a := range m.Authority {
		a.WriteTo(w)
	}
	for _, a := range m.Additional {
		a.WriteTo(w)
	}
	w.Flush()
	return b.Bytes()
}

func (m Message) labelAtOffset(targetOffset int) (DomainName, bool) {
	curr := m.Header.offset()
	for _, q := range m.Questions {
		subCurr := 0
		for i, l := range q.QName.labels {
			if curr+subCurr == targetOffset {
				lab := q.QName.labels[i:]
				return DomainName{
					labels: lab,
				}, true
			}
			subCurr += int(l.len())
		}
		curr += q.offset()
	}
	for _, a := range m.Answers {
		subCurr := 0
		for i, l := range a.Name.labels {
			if curr+subCurr == targetOffset {
				return DomainName{
					labels: a.Name.labels[i:],
				}, true
			}
			subCurr += int(l.len())
		}

		// no need to check rdata if its an ip
		if a.Type == RecordTypeA {
			curr += a.offset()
			break
		}
		subCurr = a.offsetWithoutRData()
		for i, l := range a.RData.labels {
			if curr+subCurr == targetOffset {
				return DomainName{
					labels: a.RData.labels[i:],
				}, true
			}
			subCurr += int(l.len())
		}
		curr += a.offset()
	}
	for _, a := range m.Authority {
		if curr == targetOffset {
			return a.Name, true
		}
		if curr+a.offsetWithoutRData() == targetOffset {
			return a.RData, true
		}
		subCurr := a.offsetWithoutRData()
		for i, l := range a.RData.labels {
			if l.isOffset() {
				break
			}
			subCurr += int(l.len())
			if curr+subCurr == targetOffset {
				return DomainName{
					labels: a.RData.labels[i+1:],
				}, true
			}
		}
		curr += a.offset()
	}
	for _, a := range m.Additional {
		if curr == targetOffset {
			return a.Name, true
		}
		curr += a.offset()
	}

	return DomainName{}, false
}

func (m Message) fullNameOfRecord(a ResourceRecord) string {
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

func (m Message) fullRDataOfRecord(a ResourceRecord) string {
	var sb strings.Builder
	if a.Type == RecordTypeA {
		ipBuf := []byte(a.RData.labels[0].str)
		return fmt.Sprintf("%d.%d.%d.%d",
			ipBuf[0],
			ipBuf[1],
			ipBuf[2],
			ipBuf[3],
		)
	} else {
		cur := a.RData
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
}

func (m Message) RecordsOfDomainName(dns string) []ResourceRecord {
	results := []ResourceRecord{}

	for _, a := range m.Answers {
		fullName := m.fullNameOfRecord(a)
		if fullName == dns {
			results = append(results, a)
		}
	}
	for _, a := range m.Additional {
		fullName := m.fullNameOfRecord(a)
		if fullName == dns {
			results = append(results, a)
		}
	}

	return results
}
