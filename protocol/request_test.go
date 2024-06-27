package protocol

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"testing"
)

func TestEncodeHostName(t *testing.T) {
	want := string(byte(3)) + "dns" + string(byte(6)) + "google" + string(byte(3)) + "com" + string(byte(0))
	name := "dns.google.com"

	got := string(EncodeHostName(name))
	if want != got {
		t.Fatalf("want: %s, got: %s", want, got)
	}
}

func TestMessageEncodedToHex(t *testing.T) {
	want := "00160100000100000000000003646e7306676f6f676c6503636f6d0000010001"

	req := NewRequest(
		WithQuestion("dns.google.com", 1, 1),
		WithRecursionDesired(),
		WithID(22),
	)

	got := string(req.Encode())

	if want != got {
		t.Fatalf("want\n%s\ngot\n%s", want, got)
	}
}

func TestResponseFirstTwoBytes(t *testing.T) {
	want := "00"

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
		fmt.Println("error writing to server: ", err)
	}
	err = rw.Flush()

	recv := make([]byte, 2)
	_, err = rw.Read(recv)
	if err != nil {
		fmt.Println("error listening to server: ", err)
	}
	got := string(recv)

	if want != got {
		t.Fatalf("want\n%s\ngot\n%s", want, got)
	}
}
