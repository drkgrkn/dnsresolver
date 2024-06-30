package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

const (
	offsetFlagExcess = 0b11000000 << 8
)

func encodeString(s string) []byte {
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
	i := 0
	for {
		lenLabel := int(b[i])
		for j := i + 1; j <= i+lenLabel; j++ {
			if j >= len(b) {
				return "", fmt.Errorf("given label length is larger than hostname, want to index: %d, hostname length: %d", j, len(b))
			}
			sb.WriteByte(b[j])
		}
		i += lenLabel + 1
		if i >= len(b) {
			return "", fmt.Errorf("index larger than buffer length, index: %d, buffer length: %d", i, len(b))
		}
		if b[i] == 0 {
			break
		}
		sb.WriteByte('.')
	}
	return sb.String(), nil
}

func parseHeader(r io.Reader) (Header, error) {
	// id
	idBuf, err := read(r, 2)
	if err != nil {
		return Header{}, fmt.Errorf("reading id: %w", err)
	}
	id := binary.BigEndian.Uint16(idBuf)

	// flags
	flagsBuf, err := read(r, 2)
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

func parseQuestion(r io.Reader) (Question, error) {
	// qname
	qName, err := parseDomainName(r)
	if err != nil {
		return Question{}, fmt.Errorf("error reading name: %w", err)
	}

	// counts
	countsBuf, err := read(r, 4)
	if err != nil {
		return Question{}, err
	}
	qType := binary.BigEndian.Uint16(countsBuf)
	qClass := binary.BigEndian.Uint16(countsBuf[2:])
	return Question{
		QName:  qName,
		QType:  qType,
		QClass: qClass,
	}, nil
}

func parseQuestions(r io.Reader, n int) ([]Question, error) {
	questions := make([]Question, 0, n)
	for range n {
		q, err := parseQuestion(r)
		if err != nil {
			return questions, err
		}
		questions = append(questions, q)
	}

	return questions, nil
}

func parseAnswer(r io.Reader) (Answer, error) {
	// name
	domainName, err := parseDomainName(r)
	if err != nil {
		return Answer{}, fmt.Errorf("error reading name: %w", err)
	}

	// type
	kindBuf, err := read(r, 2)
	if err != nil {
		return Answer{}, err
	}
	kind := binary.BigEndian.Uint16(kindBuf)

	// class
	classBuf, err := read(r, 2)
	if err != nil {
		return Answer{}, err
	}
	class := binary.BigEndian.Uint16(classBuf)

	// ttl
	ttlBuf, err := read(r, 4)
	if err != nil {
		return Answer{}, err
	}
	ttl := binary.BigEndian.Uint32(ttlBuf)

	// rdLength
	rdLengthBuf, err := read(r, 2)
	if err != nil {
		return Answer{}, err
	}
	rdLength := binary.BigEndian.Uint16(rdLengthBuf)

	// rdata
	rdata := make([]byte, 0)
	if kind == RecordTypeA && class == RecordClassIN {
		// A Record
		rdataBuf, err := read(r, 4)
		if err != nil {
			return Answer{}, fmt.Errorf("parsing rdata: %w", err)
		}
		rdata = rdataBuf

	} else if kind == RecordTypeNS && class == RecordClassIN {
		// NS
		//rdataBuf, _, err := parseName(r)
	} else {
		return Answer{}, fmt.Errorf(
			"unsupported record (type,class) combination, type: %d, class: %d",
			kind,
			class,
		)
	}

	return Answer{
		Name:     domainName,
		Type:     kind,
		Class:    class,
		TTL:      ttl,
		RDLength: rdLength,
		RData:    rdata,
	}, nil
}

func parseAnswers(r io.Reader, n int) ([]Answer, error) {
	answers := make([]Answer, 0, n)
	for range n {
		q, err := parseAnswer(r)
		if err != nil {
			return answers, err
		}
		answers = append(answers, q)
	}

	return answers, nil
}

func Parse(r io.Reader) (Message, error) {
	header, err := parseHeader(r)
	if err != nil {
		return Message{}, fmt.Errorf("reading header: %w", err)
	}
	questions, err := parseQuestions(r, int(header.QDCount))
	if err != nil {
		return Message{}, fmt.Errorf("reading questions: %w", err)
	}
	answers, err := parseAnswers(r, int(header.ANCount))
	if err != nil {
		return Message{}, fmt.Errorf("reading answers: %w", err)
	}
	m := Message{
		Header:    header,
		Questions: questions,
		Answers:   answers,
	}
	m.msg = m.encode()

	return m, nil
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
