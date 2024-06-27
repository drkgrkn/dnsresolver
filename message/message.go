package message

import (
	"encoding/binary"
	"encoding/hex"
	"strings"
)

func UInt16ToByteSlice(u uint16) []byte {
	arr := make([]byte, 2)
	binary.BigEndian.PutUint16(arr[0:2], u)
	return arr
}

type Flags uint16

func (f Flags) RecursionDesired(recursionDesired bool) Flags {
	mask := Flags(0x0100)
	if recursionDesired {
		f |= mask
	} else {
		mask = ^mask
		f &= mask
	}

	return f
}

type Message struct {
	ID      uint16
	Flags   Flags
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
	QName   []byte
	QType   uint16
	QClass  uint16
}

func New(hostName string) Message {
	return Message{
		ID:      22,
		Flags:   Flags(0).RecursionDesired(true),
		QDCount: 1,
		ANCount: 0,
		NSCount: 0,
		ARCount: 0,
		QName:   []byte(EncodeHostName(hostName)),
		QType:   1,
		QClass:  1,
	}

}

func (m Message) EncodeToHex() []byte {
	dst := make([]byte, 0)
	dst = hex.AppendEncode(dst, UInt16ToByteSlice(m.ID))
	dst = hex.AppendEncode(dst, UInt16ToByteSlice(uint16(m.Flags)))
	dst = hex.AppendEncode(dst, UInt16ToByteSlice(m.QDCount))
	dst = hex.AppendEncode(dst, UInt16ToByteSlice(m.ANCount))
	dst = hex.AppendEncode(dst, UInt16ToByteSlice(m.NSCount))
	dst = hex.AppendEncode(dst, UInt16ToByteSlice(m.ARCount))
	dst = hex.AppendEncode(dst, m.QName)
	dst = hex.AppendEncode(dst, UInt16ToByteSlice(m.QType))
	dst = hex.AppendEncode(dst, UInt16ToByteSlice(m.QClass))

	return dst
}

func Encode(s string) []byte {
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
