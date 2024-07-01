package protocol

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"reflect"
	"slices"
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

func writeReqReadResp(rw *bufio.ReadWriter, req Message) (Message, error) {
	n, err := rw.Write(req.Bytes())
	if err != nil {
		return Message{}, err
	}
	if n != len(req.Bytes()) {
		return Message{}, fmt.Errorf("wrote %d bytes but message is %d bytes long", n, len(req.Bytes()))
	}
	err = rw.Flush()
	if err != nil {
		return Message{}, err
	}

	resp, err := Parse(rw)

	return resp, nil
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

	resp, err := writeReqReadResp(rw, req)
	if err != nil {
		t.Fatalf("error while sending/reading request %s", err)
	}

	got := resp.Header.ID

	if want != int(got) {
		t.Fatalf("expected msg encoding to look like\n%v\nbut got\n%v\n", want, got)
	}
}

func Test_step3(t *testing.T) {
	var (
		want = []string{
			"8.8.4.4",
			"8.8.8.8",
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

	resp, err := writeReqReadResp(rw, req)
	if err != nil {
		t.Fatalf("error while sending/reading request %s", err)
	}

	records := resp.RecordsOfDomainName(target)
	got := []string{}
	for _, rec := range records {
		ipBuf := rec.formattedRData()
		got = append(got, fmt.Sprintf("%d.%d.%d.%d",
			ipBuf[0],
			ipBuf[1],
			ipBuf[2],
			ipBuf[3],
		))
	}
	slices.Sort(got)

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("expected results ips to be\n%v\nbut got\n%v\n", want, got)
	}
}

func Test_step4(t *testing.T) {
	var (
		want = []string{
			"8.8.4.4",
			"8.8.8.8",
		}

		target = "dns.google.com"
		ip     = net.IPv4(198, 41, 0, 4)
		port   = 53
	)
	req := NewMessage(
		WithID(22),
		WithQuestion(target, 1, 1),
	)

	for {
		addr := &net.UDPAddr{
			IP:   ip,
			Port: port,
			Zone: "",
		}

		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			log.Fatalf("error dialing %s", err)
		}
		defer conn.Close()
		rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

		resp, err := writeReqReadResp(rw, req)
		if err != nil {
			log.Fatalf("error while sending/reading request %s", err)
		}

		results := resp.RecordsOfDomainName("dns.google.com")
		if len(results) == 0 {
			nextRoot := resp.Authority[0]
			fullName := resp.fullRDataOfRecord(nextRoot)
			adds := resp.RecordsOfDomainName(fullName)
			for _, a := range adds {
				if a.Type == RecordTypeA {
					ip = net.IP(a.formattedRData())
					break
				}
			}
		} else {
			got := []string{}
			for _, rec := range results {
				ipBuf := rec.formattedRData()
				got = append(got, fmt.Sprintf("%d.%d.%d.%d",
					ipBuf[0],
					ipBuf[1],
					ipBuf[2],
					ipBuf[3],
				))
			}
			slices.Sort(got)

			if !reflect.DeepEqual(want, got) {
				t.Fatalf("expected results ips to be\n%v\nbut got\n%v\n", want, got)
			}
			break
		}

	}
}
