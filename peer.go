package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/golang/protobuf/proto"
	"github.com/learnscalability/gossip/pb"
)

// PeerConfig is used to unpack configurations from a config file.
type PeerConfig struct {
	// Peer Unique Identifier.
	Pid string `json:"pid"`
	// The address where the peer sends datagrams from.
	Local string `json:"local"`
	// The address where the peer listens for datagrams.
	Bind string `json:"bind"`
	// HTTP interface used to control the peer.
	CmdBind string `json:"cmdbind,omitempty"`
	// Initial list of known peers.
	Peers []PeerConfig `json:"peers,omitempty"`
}

// Peer is the peer running this process.
type Peer struct {
	config       *PeerConfig
	view         *View
	saddr, laddr *net.UDPAddr
	listener     *net.UDPConn
	cmdServer    http.Server
}

func NewPeer(cfg io.Reader) (*Peer, error) {
	var (
		err    error
		config PeerConfig
		peer   Peer
		mux    *http.ServeMux
		view   *View
	)
	err = json.NewDecoder(cfg).Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse json peer config file with error: %+v", err)
	}
	peer.config = &config
	// Setting up UDP listener
	peer.saddr, err = net.ResolveUDPAddr("udp", config.Bind)
	if err != nil {
		return nil, fmt.Errorf("Failed to prepare address to bind to UDP: %+v", err)
	}
	peer.laddr, err = net.ResolveUDPAddr("udp", config.Local)
	if err != nil {
		return nil, fmt.Errorf("Failed to prepare local bind address to source datagrams: %+v", err)
	}
	// Setting up the http command endpoints.
	mux = http.NewServeMux()
	mux.HandleFunc("/send", peer.sendHandler)
	peer.cmdServer = http.Server{
		Addr:    config.CmdBind,
		Handler: mux,
	}
	// Setting up the peers connection
	view, err = NewView(peer.laddr, config.Peers)
	if err != nil {
		return nil, fmt.Errorf("Failed to construct the set of peers in the view: %+v", err)
	}
	peer.view = view
	return &peer, nil
}

// Listen starts both the UDP listener and the HTTP command server.
func (p *Peer) Listen() error {
	var (
		err error
	)
	// start UDP listener.
	p.listener, err = net.ListenUDP("udp", p.saddr)
	if err != nil {
		return fmt.Errorf("Failed to start UDP server with error: %+v", err)
	}
	log.Printf("UDP peer listening on `%s` and publishing from `%s`\n", p.config.Bind, p.config.Local)
	go p.udpHandler()
	// start http command server.
	go func() {
		log.Printf("HTTP command server started on %s\n", p.config.CmdBind)
		p.cmdServer.ListenAndServe()
	}()
	return nil
}

func (p *Peer) udpHandler() {
	var (
		n      int
		caddr  *net.UDPAddr
		err    error
		buf    = make([]byte, 10*1024) // 10KB
		update pb.Update
	)
	for {
		n, caddr, err = p.listener.ReadFromUDP(buf)
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
		rp     *RemotePeer
		err    error
		update pb.Update
		buf    []byte
	)
	rp, err = p.view.Get(sp.Pid)
	if err != nil {
		return fmt.Errorf("Peer with Pid %s not found: %+v", sp.Pid, err)
	}
	update = pb.Update{
		Payload: []byte(sp.Content),
	}
	buf, err = proto.Marshal(&update)
	if err != nil {
		return fmt.Errorf("Failed to marshall update %+v with error: %+v", sp, err)
	}
	_, err = rp.conn.Write(buf)
	if err != nil {
		return fmt.Errorf("Failed to publish update %+v with error: %+v", sp, err)
	}
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
	p.view.Cleanup()
}

func main() {
	var (
		err  error
		sigs = make(chan os.Signal)
		sig  os.Signal
		fp   *os.File
		peer *Peer
	)
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("Expected first argument to be a config file")
	}
	fp, err = os.Open(flag.Arg(0))
	if err != nil {
		log.Fatalf("Failed to open file `%d` with error: %+v", flag.Arg(0), err)
	}
	peer, err = NewPeer(fp)
	if err != nil {
		log.Fatal(err)
	}
	err = peer.Listen()
	if err != nil {
		log.Fatal(err)
	}
	defer peer.Close()
	// Setup termination handlers.
	signal.Notify(sigs, os.Interrupt, os.Kill)
	sig = <-sigs
	log.Fatalf("Received signal %+v. Terminating", sig)
}
