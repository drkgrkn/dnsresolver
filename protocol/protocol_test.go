package protocol

import (
	"bufio"
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

func Test_step1(t *testing.T) {
	want, _ := hex.DecodeString("00160100000100000000000003646e7306676f6f676c6503636f6d0000010001")

	req := NewMessage(
		WithID(22),
		WithRecursionDesired(),
		WithQuestion("dns.google.com", 1, 1),
	)

	got := req.Bytes()

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("expected msg encoding to look like\n%v\nbut got\n%v\n", want, got)
	}
}

func Test_step2(t *testing.T) {
	var (
		want = 22

		ip   = net.IPv4(8, 8, 8, 8)
		port = 53
	)
	addr := &net.UDPAddr{
		IP:   ip,
		Port: port,
		Zone: "",
	}

	conn, err := net.DialUDP("udp", nil, addr)
	defer conn.Close()
	if err != nil {
		log.Fatalf("error dialing google")
	}
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	req := NewMessage(
		WithID(22),
		WithRecursionDesired(),
		WithQuestion("dns.google.com", 1, 1),
	)

	n, err := rw.Write(req.Bytes())
	if err != nil {
		log.Fatal("error writing to server: ", err)
	}
	if n != len(req.Bytes()) {
		log.Fatalf("couldnt write entire message, msg len: %d, written len: %d", len(req.Bytes()), n)
	}
	err = rw.Flush()

	resp, err := Parse(rw)
	if err != nil {
		log.Fatalf("error decoding resp: %s", err)
	}

	got := resp.Header.ID

	if want != int(got) {
		t.Fatalf("expected msg encoding to look like\n%v\nbut got\n%v\n", want, got)
	}
}

func Test_step3(t *testing.T) {
	var (
		want = []string{
			"8.8.8.8",
			"8.8.4.4",
		}

		target = "dns.google.com"
		ip     = net.IPv4(8, 8, 8, 8)
		port   = 53
	)
	addr := &net.UDPAddr{
		IP:   ip,
		Port: port,
		Zone: "",
	}

	conn, err := net.DialUDP("udp", nil, addr)
	defer conn.Close()
	if err != nil {
		log.Fatalf("error dialing google")
	}
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	req := NewMessage(
		WithID(22),
		WithRecursionDesired(),
		WithQuestion(target, 1, 1),
	)

	n, err := rw.Write(req.Bytes())
	if err != nil {
		log.Fatal("error writing to server: ", err)
	}
	if n != len(req.Bytes()) {
		log.Fatalf("couldnt write entire message, msg len: %d, written len: %d", len(req.Bytes()), n)
	}
	err = rw.Flush()

	resp, err := Parse(rw)
	if err != nil {
		log.Fatalf("error decoding resp: %s", err)
	}

	got := resp.RecordsOfDomainName(target)

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("expected results ips to be\n%v\nbut got\n%v\n", want, got)
	}
}
