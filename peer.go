package main

// See: http://www.minaandrawos.com/2016/05/14/udp-vs-tcp-in-golang/

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

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
	cmdServer http.Server
}

func NewPeer(cfg io.Reader) (*Peer, error) {
	var (
		err    error
		config PeerConfig
		peer   Peer
		mux    *http.ServeMux
	)
	err = json.NewDecoder(cfg).Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse json peer config file with error: %+v", err)
	}
	peer.config = &config
	// Setting up the http command endpoints.
	mux = http.NewServeMux()
	mux.HandleFunc("/send", peer.sendHandler)
	peer.cmdServer = http.Server{
		Addr:    config.CmdBind,
		Handler: mux,
	}
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
	go func() {
		log.Printf("HTTP command server started on %s\n", p.config.CmdBind)
		p.cmdServer.ListenAndServe()
	}()
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

type SendPayload struct {
	Pid     string `json:"pid"`
	Content string `json:"content"`
}

func (p *Peer) Send(sp *SendPayload) error {
	var (
		buf    []byte
		update pb.Update
		err    error
		pc     PeerConfig
		ok     bool
		conn   net.Conn
	)
	update = pb.Update{
		Payload: []byte(sp.Content),
	}
	buf, err = proto.Marshal(&update)
	if err != nil {
		return fmt.Errorf("Failed to marshall update %+v with error: %+v", sp, err)
	}
	if pc, ok = p.view[sp.Pid]; !ok {
		return fmt.Errorf("Could not find remote peer with pid %s", sp.Pid)
	}
	conn, err = net.Dial("udp", pc.Bind)
	if err != nil {
		return fmt.Errorf("Failed to contact remote peer %+v with error: %+v", pc, err)
	}
	defer conn.Close()
	_, err = conn.Write(buf)
	if err != nil {
		return fmt.Errorf("Failed to publish update %+v with error: %+v", sp, err)
	}
	log.Printf("Published update %+v to remote peer %+v", sp, pc)
	return nil
}

// sendHandler expects a body of the following format: {pid: string, content: string}
func (p *Peer) sendHandler(w http.ResponseWriter, r *http.Request) {
	var (
		sp  SendPayload
		err error
	)
	err = json.NewDecoder(r.Body).Decode(&sp)
	if err != nil {
		log.Println("Failed to read request body with error: %+v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	err = p.Send(&sp)
	if err != nil {
		log.Printf("Failed to send message to peer %+v with error %+v", sp, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Close closes both the UDP listener and the HTTP command server.
func (p *Peer) Close() {
	p.listener.Close()
	p.cmdServer.Close()
}
