package main

import (
	"log"
	"net"
)

func main() {
	c, err := net.Dial("unix", "/var/run/eventd_events")
	if err != nil {
		log.Fatal("Dial error", err)
	}
	defer c.Close()

	msg := "mtqq-bridge[1]: MSG_RECVD: Hello Bob"
	_, err = c.Write([]byte(msg))
	if err != nil {
		log.Fatal("Write error:", err)
	}

}
