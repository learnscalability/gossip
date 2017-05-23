package main

// See: http://www.minaandrawos.com/2016/05/14/udp-vs-tcp-in-golang/

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/golang/protobuf/proto"
	"github.com/learnscalability/gossip/pb"
)

// PeerConfig is used to unpack configurations from a config file.
type PeerConfig struct {
	// Peer Unique Identifier.
	Pid string `json:"pid"`
	// The address where the peer listens for datagrams.
	Bind string `json:"bind"`
	// HTTP interface used to control the peer.
	CmdBind string `json:"cmdbind,omitempty"`
	// Initial list of known peers.
	Peers []PeerConfig `json:"peers,omitempty"`
}

// Peer is the peer running this process.
type Peer struct {
	config    *PeerConfig
	view      View
	listener  net.PacketConn
	cmdServer *CmdServer
}

func NewPeer(cfg io.Reader) (*Peer, error) {
	var (
		err    error
		config PeerConfig
		peer   Peer
	)
	err = json.NewDecoder(cfg).Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse json peer config file with error: %+v", err)
	}
	peer.config = &config
	// Setting up the http command endpoints.
	peer.cmdServer = NewCmdServer(&peer)
	// Setting up the peers connection
	peer.view = NewView(config.Peers)
	return &peer, nil
}

// Listen starts both the UDP listener and the HTTP command server.
func (p *Peer) Listen() error {
	var (
		err      error
		listener net.PacketConn
	)
	// Setting up UDP listener
	listener, err = net.ListenPacket("udp", p.config.Bind)
	if err != nil {
		return fmt.Errorf("Unable to bind to udp address %s with error: %+v", p.config.Bind, err)
	} else {
		log.Printf("UDP peer started on %s and accepting connections on %v", p.config.Bind, listener.LocalAddr())
	}
	p.listener = listener
	go p.udpHandler()
	// start http command server.
	go p.cmdServer.Run()
	return nil
}

// udpHandler read datagrams that are comming through the pipes.
// Should be run as a goroutine.
func (p *Peer) udpHandler() {
	var (
		n      int
		caddr  net.Addr
		err    error
		buf    = make([]byte, 10*1024) // 10KB
		update pb.Update
	)
	for {
		n, caddr, err = p.listener.ReadFrom(buf)
		if err != nil {
			log.Fatalf("Failed to read datagram: %T %+v", err, err)
		}
		err = proto.Unmarshal(buf[:n], &update)
		if err != nil {
			log.Fatalf("Unable to unmarshal update data: %+v", err)
		}
		log.Printf("Received data `%s` from %s\n", update.Payload, caddr)
	}
}

func (p *Peer) Send(pid string, content string) error {
	var (
		buf    []byte
		update pb.Update
		err    error
		pc     PeerConfig
		ok     bool
		conn   net.Conn
	)
	update = pb.Update{
		Payload: []byte(content),
	}
	buf, err = proto.Marshal(&update)
	if err != nil {
		return fmt.Errorf("Failed to marshall update %+s with error: %+v", update, err)
	}
	if pc, ok = p.view[pid]; !ok {
		return fmt.Errorf("Could not find remote peer with pid %s", pid)
	}
	conn, err = net.Dial("udp", pc.Bind)
	if err != nil {
		return fmt.Errorf("Failed to contact remote peer %+v with error: %+v", pc, err)
	}
	defer conn.Close()
	_, err = conn.Write(buf)
	if err != nil {
		return fmt.Errorf("Failed to publish update %+s with error: %+v", content, err)
	}
	log.Printf("Published update %+v to remote peer %+s", pid, pc)
	return nil
}

// Close closes both the UDP listener and the HTTP command server.
func (p *Peer) Close() {
	p.listener.Close()
	p.cmdServer.Close()
}
