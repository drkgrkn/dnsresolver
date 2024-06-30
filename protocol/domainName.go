package protocol

import (
	"bytes"
	"io"
	"strings"
)

type Label struct {
	length uint16
	str    string
}

func (l Label) isZero() bool {
	return l.length == 0
}

func (l Label) isOffset() bool {
	return l.length >= offsetFlagExcess
}

func (l Label) offset() int {
	return int(l.length) - 0b11000000<<8
}

func (l Label) len() int {
	if l.isOffset() {
		return 2
	}

	return 2 + len(l.str)
}

func (l Label) Bytes() []byte {
	if l.isZero() {
		return []byte{0}
	}
	if l.isOffset() {
		return UInt16ToByteSlice(l.length)
	}
	lByte := []byte{byte(l.length)}
	return append(lByte, []byte(l.str)...)
}

type DomainName struct {
	labels []Label
}

func newDomainName(dn string) DomainName {
	labels := make([]Label, 0)
	words := strings.Split(dn, ".")

	for _, word := range words {
		labels = append(
			labels,
			Label{
				length: uint16(len(word)),
				str:    word,
			},
		)
	}
	labels = append(labels, Label{
		length: 0,
		str:    "",
	})

	return DomainName{
		labels: labels,
	}
}

func parseDomainName(r io.Reader) (DomainName, error) {
	labels := make([]Label, 0)

	lengthBuf := make([]byte, 1)
	for {
		_, err := r.Read(lengthBuf)
		if err != nil {
			return DomainName{}, err
		}

		// name pointer
		if lengthBuf[0] >= 0b11000000 {
			labelLength := uint16(lengthBuf[0]) << 8
			_, err := r.Read(lengthBuf)
			if err != nil {
				return DomainName{}, err
			}
			labelLength += uint16(lengthBuf[0])
			labels = append(labels, Label{
				length: labelLength,
				str:    "",
			})
			break
		}

		// normal label
		if lengthBuf[0] == 0 {
			labels = append(labels, Label{
				length: 0,
				str:    "",
			})
			break
		}
		labelBuff := make([]byte, lengthBuf[0])
		_, err = r.Read(labelBuff)
		if err != nil {
			return DomainName{}, err
		}
		newLabel := Label{
			length: uint16(lengthBuf[0]),
			str:    string(labelBuff),
		}
		labels = append(labels, newLabel)
	}

	return DomainName{
		labels: labels,
	}, nil
}

func (dn DomainName) len() int {
	length := 0
	for _, l := range dn.labels {
		length += l.len()
	}

	return length
}

func (dn DomainName) Bytes() []byte {
	var b bytes.Buffer
	for _, l := range dn.labels {
		b.Write(l.Bytes())
	}
	return b.Bytes()
}
