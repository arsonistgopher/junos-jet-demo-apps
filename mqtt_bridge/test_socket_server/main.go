package main

import (
	"fmt"
	"log"
	"net"
	"syscall"
)

func echoServer(c net.Conn) {
	for {
		buf := make([]byte, 512)
		nr, err := c.Read(buf)
		if err != nil {
			return
		}

		data := buf[0:nr]
		println("Server got:", string(data))
		_, err = c.Write(data)
		if err != nil {
			log.Fatal("Writing client error: ", err)
		}
	}
}

func main() {
	log.Println("Starting echo server")
	syscall.Unlink("/var/run/eventd_events")
	laddr := net.UnixAddr{Name: "/var/run/eventd_events", Net: "unixgram"}
	conn, err := net.ListenUnixgram("unixgram", &laddr)
	if err != nil {
		log.Fatal("Listen error: ", err)
	}

	for {
		var buf [1024]byte
		n, err := conn.Read(buf[:])
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s\n", string(buf[:n]))
	}
}
