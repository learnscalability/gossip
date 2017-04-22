package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/learnscalability/gossip/pb"
)

type PeerConfig struct {
	Pid     string       `json:"pid"`
	Local   string       `json:"local"`
	Bind    string       `json:"bind"`
	CmdBind string       `json:"cmdbind,omitempty"`
	Peers   []PeerConfig `json:"peers,omitempty"`
}

type Peer struct {
	config       *PeerConfig
	saddr, laddr *net.UDPAddr
	listener     *net.UDPConn
	cmdServer    http.Server
}

func NewPeer(cfg io.Reader) (*Peer, error) {
	var (
		err  error
		config PeerConfig
		peer Peer
		mux  *http.ServeMux
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
	mux = http.NewServeMux()
	mux.HandleFunc("/send", peer.commandHandler)
	peer.cmdServer = http.Server{
		Addr:    config.CmdBind,
		Handler: mux,
	}
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

func (p *Peer) commandHandler(w http.ResponseWriter, r *http.Request) {
	var (
		body   []byte
		paddr  *net.UDPAddr
		err    error
		conn   *net.UDPConn
		update pb.Update
		i      int
		buf    []byte
	)
	body, err = ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("Failed to read request body with error: %+v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	paddr, err = net.ResolveUDPAddr("udp", string(body))
	if err != nil {
		log.Println("Failed to prepare peer address to send datagrams to: %+v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	conn, err = net.DialUDP("udp", p.laddr, paddr)
	if err != nil {
		log.Printf("Failed to connect to UDP server: %+v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	for i = 1; i <= 10; i++ {
		update = pb.Update{
			Payload: []byte(strconv.Itoa(i)),
		}
		buf, err = proto.Marshal(&update)
		if err != nil {
			log.Fatalf("Failed to marshall update %d with error: %+v", i, err)
		}
		_, err = conn.Write(buf)
		if err != nil {
			log.Fatalf("Failed to publish datagram %d with error: %+v", i, err)
		}
	}
}

// Close closes both the UDP listener and the HTTP command server.
func (p *Peer) Close() {
	p.listener.Close()
	p.cmdServer.Close()
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
