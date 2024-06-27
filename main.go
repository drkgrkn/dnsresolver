package main

import (
	"bufio"
	"fmt"
	"log"
	"net"

	"github.com/drkgrkn/dnsresolver/message"
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
	if err != nil {
		log.Fatalf("error dialing google")
	}
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))

	msg := message.New("dns.google.com")

	msgEncoded := msg.EncodeToHex()
	_, err = rw.Write(msgEncoded)
	if err != nil {
		fmt.Println("error writing to server: ", err)
	}
	err = rw.Flush()
	log.Println("wrote to google")

	recv := make([]byte, 2)
	_, err = rw.Read(recv)
	if err != nil {
		fmt.Println("error listening to server: ", err)
	}
	log.Println("read from google")

	fmt.Println(recv)
	fmt.Printf("%s\n", string(recv))
}
