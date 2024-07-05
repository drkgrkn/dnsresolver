package protocol

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

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

func Find(target string) []string {
	var (
		ip   = net.IPv4(198, 41, 0, 4)
		port = 53
	)
	for {
		req := NewMessage(
			WithID(22),
			WithQuestion(target, 1, 1),
		)

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

	outer:
		for {
			var (
				ans          = make([]string, 0)
				scopedTarget = target
			)

			// result was found
			if len(resp.Answers) != 0 {
				for _, rr := range resp.Answers {
					switch rr.Type {
					case RecordTypeAAAA:
						break
					case RecordTypeCNAME:
						if scopedTarget == resp.fullNameOfRecord(rr) {
							scopedTarget = resp.fullRDataOfRecord(rr)
						}
					case RecordTypeA:
						ans = append(ans, resp.fullRDataOfRecord(rr))
					}
				}
				return ans
			} else {
				for _, rr := range resp.Authority {
					additionals := resp.RecordsOfDomainName(resp.fullRDataOfRecord(rr))
					if len(additionals) != 0 {
						for _, additional := range additionals {
							if additional.Type == RecordTypeA {
								ip = net.IP(resp.formattedRDataOf(additional))
								break outer
							}
						}
					}
				}
			}
		}
	}
}
