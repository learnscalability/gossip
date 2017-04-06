package main

import (
	"flag"
	"log"
	"net"
	"strconv"
)

func main() {
	var (
		server, local string
		saddr, laddr *net.UDPAddr
		err error
		conn *net.UDPConn
		i int
		buf []byte
	)
	flag.StringVar(&server, "server", ":3000", "server ip:port")
	flag.StringVar(&local, "local", ":4000", "local source ip:port")
	flag.Parse()
	saddr, err = net.ResolveUDPAddr("udp", server)
	if err != nil {
		log.Fatalf("Failed to server address to contact: %+v", err)
	}
	laddr, err = net.ResolveUDPAddr("udp", local)
	if err != nil {
		log.Fatalf("Failed to prepare local address to source datagrams: %+v", err)
	}
	conn, err = net.DialUDP("udp", laddr, saddr)
	if err != nil {
		log.Fatalf("Failed to connect to UDP server: %+v", err)
	}
	defer conn.Close()
	for i = 1; i <= 10; i++ {
		buf = []byte(strconv.Itoa(i))
		_, err = conn.Write(buf)
		if err != nil {
			log.Fatalf("Failed to publish datagram %d with error: %+v", i, err)
		}
	}
}
