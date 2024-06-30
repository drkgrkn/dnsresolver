package protocol

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"reflect"
	"strings"
	"testing"
)

func toHexUint16(values ...uint16) string {
	var sb strings.Builder
	for _, u := range values {
		hx := fmt.Sprintf("%02x", u)
		sb.WriteString(hx)
	}
	return sb.String()
}

func TestEncodeHostName(t *testing.T) {
	want := string(byte(3)) + "dns" + string(byte(6)) + "google" + string(byte(3)) + "com" + string(byte(0))
	name := "dns.google.com"

	got := string(encodeHostName(name))
	if want != got {
		t.Fatalf("want: %s, got: %s", want, got)
	}
}
func TestReadHeader(t *testing.T) {
	want := Header{
		ID:      22,
		Flags:   5,
		QDCount: 3,
		ANCount: 2,
		NSCount: 1,
		ARCount: 0,
	}

	inputStream := []byte{}
	inputStream = append(
		inputStream,
		0, 22, 0, 5, 0, 3, 0, 2, 0, 1, 0, 0)

	got, err := parseHeader(bytes.NewReader(inputStream))
	if err != nil {
		t.Fatalf("error while reading stream: %s", err)
	}

	if want.ID != got.ID {
		t.Fatalf("reading header id:\nwant\n%d\ngot\n%d", want.ID, got.ID)
	}
	if want.Flags != got.Flags {
		t.Fatalf("reading header flags:\nwant\n%d\ngot\n%d", want.Flags, got.Flags)
	}
	if want.ANCount != got.ANCount {
		t.Fatalf("reading header ancount:\nwant\n%d\ngot\n%d", want.ANCount, got.ANCount)
	}
	if want.ARCount != got.ARCount {
		t.Fatalf("reading header arcount:\nwant\n%d\ngot\n%d", want.ARCount, got.ARCount)
	}
	if want.NSCount != got.NSCount {
		t.Fatalf("reading header nscount:\nwant\n%d\ngot\n%d", want.NSCount, got.NSCount)
	}
	if want.QDCount != got.QDCount {
		t.Fatalf("reading header qdcount:\nwant\n%d\ngot\n%d", want.QDCount, got.QDCount)
	}
}

func TestReadQuestion(t *testing.T) {
	want := Question{
		QName:  []byte(string(byte(3)) + "dns" + string(byte(6)) + "google" + string(byte(3)) + "com" + string(byte(0))),
		QType:  1,
		QClass: 1,
	}
	inputStream := []byte{
		3,
		'd',
		'n',
		's',
		6,
		'g',
		'o',
		'o',
		'g',
		'l',
		'e',
		3,
		'c',
		'o',
		'm',
		0,
		0,
		1,
		0,
		1,
	}
	got, err := parseQuestion(bytes.NewReader(inputStream))
	if err != nil {
		t.Fatalf("error while reading stream: %s", err)
	}
	if string(want.QName) != string(got.QName) {
		t.Fatalf("reading question name:\nwant\n%s\ngot\n%s", want.QName, got.QName)
	}
	if want.QClass != got.QClass {
		t.Fatalf("reading question class:\nwant\n%d\ngot\n%d", want.QClass, got.QClass)
	}
	if want.QType != got.QType {
		t.Fatalf("reading question type:\nwant\n%d\ngot\n%d", want.QType, got.QType)
	}

}

func TestMessageEncodedToHex(t *testing.T) {
	want := "00160100000100000000000003646e7306676f6f676c6503636f6d0000010001"

	req := NewRequest(
		WithQuestion("dns.google.com", 1, 1),
		WithRecursionDesired(),
		WithID(22),
	)

	got := hex.EncodeToString(req.Encode())

	if want != got {
		t.Fatalf("want\n%s\ngot\n%s", want, got)
	}
}

func TestResponseFirstTwoBytes(t *testing.T) {
	want := 22

	var (
		ip   = net.IPv4(8, 8, 8, 8)
		port = 53
	)
	addr := &net.UDPAddr{
		IP:   ip,
		Port: port,
		Zone: "",
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatalf("error dialing google")
	}
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	req := NewRequest(
		WithQuestion("dns.google.com", 1, 1),
		WithRecursionDesired(),
		WithID(22),
	)

	msgEncoded := req.Encode()
	_, err = rw.Write(msgEncoded)
	if err != nil {
		t.Fatal("error writing to server: ", err)
	}
	err = rw.Flush()
	if err != nil {
		t.Fatal("error writing to server: ", err)
	}

	recv := make([]byte, 2)
	_, err = rw.Read(recv)
	if err != nil {
		t.Fatal("error listening to server: ", err)
	}
	got := 0
	got += int(recv[0])<<8 + int(recv[1])

	if want != got {
		t.Fatalf("want\n%d\ngot\n%d", want, got)
	}
}

func TestGoogleResolvedToCorrect(t *testing.T) {
	var (
		ip        = net.IPv4(8, 8, 8, 8)
		port      = 53
		dnsServer = "dns.google.com"

		wantFirst  = []byte{8, 8, 8, 8}
		wantSecond = []byte{8, 8, 4, 4}
	)
	addr := &net.UDPAddr{
		IP:   ip,
		Port: port,
		Zone: "",
	}

	conn, err := net.DialUDP("udp", nil, addr)
	defer conn.Close()
	if err != nil {
		t.Fatalf("error dialing google")
	}
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	req := NewRequest(
		WithQuestion(dnsServer, 1, 1),
		WithRecursionDesired(),
		WithID(22),
	)

	msgEncoded := req.Encode()
	n, err := rw.Write(msgEncoded)
	if err != nil {
		t.Fatal("error writing to server: ", err)
	}
	if n != len(msgEncoded) {
		t.Fatalf("couldnt write entire message, msg len: %d, written len: %d", len(msgEncoded), n)
	}
	err = rw.Flush()
	if err != nil {
		t.Fatal("error writing to server: ", err)
	}

	msg, err := Parse(rw)
	if err != nil {
		t.Fatalf("error decoding resp: %s", err)
	}

	results := msg.ToMap()[dnsServer]
	for _, got := range results {
		if !reflect.DeepEqual(got, wantFirst) && !reflect.DeepEqual(got, wantSecond) {
			t.Fatalf("expected the dns to resolve to either %v or %v, but got %v", wantFirst, wantSecond, got)
		}

	}

}
