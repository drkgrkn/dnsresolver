package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
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
		dst = append(dst, a.Name...)
		dst = append(dst, UInt16ToByteSlice(a.Type)...)
		dst = append(dst, UInt16ToByteSlice(a.Class)...)
		dst = append(dst, UInt32ToByteSlice(a.TTL)...)
		dst = append(dst, UInt16ToByteSlice(a.RDLength)...)
		dst = append(dst, a.RData...)
	}
	return dst
}

func readHeader(r io.Reader) (Header, error) {
	// id
	idBuf, err := read(r, 2)
	if err != nil {
		return Header{}, fmt.Errorf("reading id: %w", err)
	}
	id := binary.BigEndian.Uint16(idBuf)

	// flags
	flagsBuf, err := read(r, 2)
	log.Println(flagsBuf)
	if err != nil {
		return Header{}, fmt.Errorf("reading flags: %w", err)
	}
	flags := binary.BigEndian.Uint16(flagsBuf)

	// counts
	countsBuf, err := read(r, 8)
	if err != nil {
		return Header{}, fmt.Errorf("reading counts: %w", err)
	}
	qdCount := binary.BigEndian.Uint16(countsBuf[0:2])
	anCount := binary.BigEndian.Uint16(countsBuf[2:4])
	nsCount := binary.BigEndian.Uint16(countsBuf[4:6])
	arCount := binary.BigEndian.Uint16(countsBuf[6:8])
	return Header{
		ID:      id,
		Flags:   flags,
		QDCount: qdCount,
		ANCount: anCount,
		NSCount: nsCount,
		ARCount: arCount,
	}, nil
}

func readQuestion(r io.Reader) (Question, error) {
	var b bytes.Buffer
	for {
		lengthBuf := make([]byte, 1)
		_, err := r.Read(lengthBuf)
		if err != nil {
			return Question{}, err
		}
		b.Write(lengthBuf)
		if lengthBuf[0] == 0 {
			break
		}
		label := make([]byte, lengthBuf[0])
		_, err = r.Read(label)
		if err != nil {
			return Question{}, err
		}
		b.Write(label)
	}

	countsBuf, err := read(r, 4)
	if err != nil {
		return Question{}, err
	}
	qType := binary.BigEndian.Uint16(countsBuf)
	qClass := binary.BigEndian.Uint16(countsBuf[2:])
	return Question{
		QName:  b.Bytes(),
		QType:  qType,
		QClass: qClass,
	}, nil
}

func readQuestions(r io.Reader, n int) ([]Question, error) {
	questions := make([]Question, 0, n)
	for range n {
		q, err := readQuestion(r)
		if err != nil {
			return questions, err
		}
		questions = append(questions, q)
	}

	return questions, nil
}

func Decode(r io.Reader) (Message, error) {
	header, err := readHeader(r)
	if err != nil {
		return Message{}, fmt.Errorf("reading header: %w", err)
	}
	questions, err := readQuestions(r, int(header.QDCount))
	if err != nil {
		return Message{}, fmt.Errorf("reading questions: %w", err)
	}
	return Message{
		Header:    header,
		Questions: questions,
		Answers:   []Answer{},
	}, nil
}

func read(r io.Reader, size int) ([]byte, error) {
	b := make([]byte, size)
	n, err := r.Read(b)
	if err != nil {
		return b, err
	}
	if size != n {
		return b, fmt.Errorf("not enough bytes read")
	}

	return b, nil
}

func EncodeString(s string) []byte {
	dst := make([]byte, 0)
	dst = append(dst, byte(len(s)))
	dst = append(dst, []byte(s)...)
	return dst
}

func encodeHostName(hostName string) []byte {
	dst := make([]byte, 0)
	words := strings.Split(hostName, ".")

	for _, word := range words {
		dst = append(dst, byte(len(word)))
		dst = append(dst, []byte(word)...)
	}
	dst = append(dst, 0)

	return dst
}

func decodeHostName(b []byte) (string, error) {
	var sb strings.Builder
	for i := 0; i < len(b); i++ {
		//lenLabel := int(b[i])

	}

	return sb.String(), nil
}
