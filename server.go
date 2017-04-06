package main

import (
	"flag"
	"log"
	"net"
)

func main() {
	var (
		bind string
		saddr, caddr *net.UDPAddr
		err error
		conn *net.UDPConn
		n int
		buf = make([]byte, 1024)
	)
	flag.StringVar(&bind, "bind", ":3000", "listen to incoming UDP packets")
	flag.Parse()
	saddr, err = net.ResolveUDPAddr("udp", bind)
	if err != nil {
		log.Fatalf("Failed to prepare address to bind to UDP: %+v", err)
	}
	conn, err = net.ListenUDP("udp", saddr)
	if err != nil {
		log.Fatalf("Failed to start UDP server: %+v", err)
	}
	defer conn.Close()
	log.Printf("UDP server started\n")
	for {
		n, caddr, err = conn.ReadFromUDP(buf)
		if err != nil {
			log.Fatalf("Failed to read datagram: %+v", err)
		}
		log.Printf("Received data `%s` from %s\n", buf[:n], caddr)
	}
}

