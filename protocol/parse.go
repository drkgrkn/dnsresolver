package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
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

func parseRecord(r io.Reader) (ResourceRecord, error) {
	// name
	domainName, err := parseDomainName(r)
	if err != nil {
		return ResourceRecord{}, fmt.Errorf("error reading name: %w", err)
	}

	// type
	kindBuf, err := read(r, 2)
	if err != nil {
		return ResourceRecord{}, err
	}
	kind := binary.BigEndian.Uint16(kindBuf)

	// class
	classBuf, err := read(r, 2)
	if err != nil {
		return ResourceRecord{}, err
	}
	class := binary.BigEndian.Uint16(classBuf) % 2
	if class != RecordClassIN {
		return ResourceRecord{}, fmt.Errorf("record class should be IN (%d) but got %d", RecordClassIN, class)
	}

	// ttl
	ttlBuf, err := read(r, 4)
	if err != nil {
		return ResourceRecord{}, err
	}
	ttl := binary.BigEndian.Uint32(ttlBuf)

	// rdLength
	rdLengthBuf, err := read(r, 2)
	if err != nil {
		return ResourceRecord{}, err
	}
	rdLength := binary.BigEndian.Uint16(rdLengthBuf)

	// rdata
	rdata := DomainName{}
	switch kind {
	case RecordTypeA:
		rdataBuf, err := read(r, int(rdLength)) // rdLength == 4
		if err != nil {
			return ResourceRecord{}, fmt.Errorf("parsing rdata: %w", err)
		}
		rdata = newIPAddress(string(rdataBuf))

	case RecordTypeNS:
		rdata, err = parseDomainName(r)
		if err != nil {
			return ResourceRecord{}, fmt.Errorf("error reading name: %w", err)
		}

	case RecordTypeCNAME:
		rdata, err = parseDomainName(r)
		if err != nil {
			return ResourceRecord{}, fmt.Errorf("error reading name: %w", err)
		}

	case RecordTypeAAAA:
		rdataBuf, err := read(r, int(rdLength)) // rdLength == 16
		if err != nil {
			return ResourceRecord{}, fmt.Errorf("parsing rdata: %w", err)
		}
		rdata = newDomainName(string(rdataBuf))

	default:
		return ResourceRecord{}, fmt.Errorf(
			"unsupported record (type,class) combination, type: %d, class: %d",
			kind,
			class,
		)
	}

	return ResourceRecord{
		Name:     domainName,
		Type:     kind,
		Class:    class,
		TTL:      ttl,
		RDLength: rdLength,
		RData:    rdata,
	}, nil
}

func parseRecords(r io.Reader, n int) ([]ResourceRecord, error) {
	answers := make([]ResourceRecord, 0, n)
	for range n {
		q, err := parseRecord(r)
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
	answers, err := parseRecords(r, int(header.ANCount))
	if err != nil {
		return Message{}, fmt.Errorf("reading answers: %w", err)
	}
	auths, err := parseRecords(r, int(header.NSCount))
	if err != nil {
		return Message{}, fmt.Errorf("reading authorities: %w", err)
	}
	adds, err := parseRecords(r, int(header.ARCount))
	if err != nil {
		return Message{}, fmt.Errorf("reading authorities: %w", err)
	}
	m := Message{
		Header:     header,
		Questions:  questions,
		Answers:    answers,
		Authority:  auths,
		Additional: adds,
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
