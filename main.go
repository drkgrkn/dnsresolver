package main

import (
	"fmt"
	"os"

	"github.com/drkgrkn/dnsresolver/protocol"
)

func main() {
	domain := os.Args[1]
	ips := protocol.Find(domain)
	fmt.Printf("IP addresses of %s\n", domain)
	for _, ip := range ips {
		fmt.Printf("  - %s\n", ip)
	}
}
