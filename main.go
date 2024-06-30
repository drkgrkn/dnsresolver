package main

import (
	"bufio"
	"fmt"
	"log"
	"net"

	"github.com/drkgrkn/dnsresolver/protocol"
)

func main() {
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
	defer conn.Close()
	if err != nil {
		log.Fatalf("error dialing google")
	}
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	req := protocol.NewMessage(
		protocol.WithQuestion("dns.google.com", 1, 1),
		protocol.WithRecursionDesired(),
		protocol.WithID(22),
	)

	n, err := rw.Write(req.Bytes())
	if err != nil {
		log.Fatal("error writing to server: ", err)
	}
	if n != len(req.Bytes()) {
		log.Fatalf("couldnt write entire message, msg len: %d, written len: %d", len(req.Bytes()), n)
	}
	err = rw.Flush()
	if err != nil {
		log.Fatal("error writing to server: ", err)
	}
	log.Println("wrote to google")

	resp, err := protocol.Parse(rw)
	if err != nil {
		log.Fatalf("error decoding resp: %s", err)
	}
	log.Println("received from google")

	fmt.Printf("%+v", resp)
}
