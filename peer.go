package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
)

func udpHandler(conn *net.UDPConn) {
	var (
		n int
		caddr *net.UDPAddr
		err error
		buf = make([]byte, 10 * 1024) // 10KB
	)
	for {
		n, caddr, err = conn.ReadFromUDP(buf)
		if err != nil {
			log.Fatalf("Failed to read datagram: %T %+v", err, err)
		}
		log.Printf("Received data `%s` from %s\n", buf[:n], caddr)
	}
}

func main() {
	var (
		bind string
		saddr *net.UDPAddr
		err error
		conn *net.UDPConn
		sigs = make(chan os.Signal)
		sig os.Signal
	)
	flag.StringVar(&bind, "bind", ":3000", "where to listen for connections")
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
	go udpHandler(conn)
	// Setup termination handlers.
	signal.Notify(sigs, os.Interrupt, os.Kill)
	sig = <-sigs
	log.Fatalf("Received signal %+v. Terminating", sig)
}


/*
import (
	"io"
	"math/rand"
	"net"

	"github.com/learnscalability/gossip/pb"
	"google.golang.org/grpc"
)

type StoredMessage struct {
	Payload []byte
	SendCount uint8
}

// Implements the Gossip interface
type Peer struct {
	ID string
	Bind string
	MaxSendTimes uint8 // param "t"
	Fanout uint8 // param "f"
	Buffer []*StoredMessage
	BufferCapacity uint8 // param "b"
	View []string
	ViewSize uint8 // param "l"
}

func NewPeer() (*Peer, error) {
	if id, err = newUUID(); err != nil {
		return nil, err
	}
	return &Peer{
		ID: id
	}, nil
}

func (p *Peer) Listen() error {
	var (
		ln  net.Listener
		err error
		srv *grpc.Server
	)
	ln, err = net.Listen("tcp", s.Bind)
	if err != nil {
		return err
	}
	srv = grpc.NewServer()
	pb.RegisterGossipServer(srv, p)
	go p.run(srv.Context())
	return srv.Serve(ln)
}

func (p *Peer) Join(contact string) error {
	join = &pb.Update{
		Known: []string{contact},
	}
	for peerAddr := range View {
		count, err := p.SendTo(peerAddr, sm)
	}
	p.View = append(p.View, contact)
}

func (p *Peer) Contact(u *pb.Update) (*pb.Response, error) {
	p.addToBuffer(u)
	return &pb.Reponse{}, nil
}

// Helpers

// run should be executed as a goroutine.
func (p *Peer) run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			for sm := range p.Buffer {
				for peerAddr := range p.randomView() {
					count, err := p.SendTo(peerAddr, sm)
					if err != nil {
					}
				}
			}
		}
	}
}

func (p *Peer) SendTo(addr string, sm *StoredMessage) (newSendCount int, err error) {
	var cl, err = p.NewClient(addr)
	if err != nil {
		return 0, err
	}
	defer cl.Close()
	var u = &pb.Update{
		Payload: sm.Payload,
		Known: p.Buffer,
	}
	if err = cl.Update(u); err != nil {
		return 0, err
	}
	return sm.SendCount + 1, nil
}

// NewClient connects to the server url given.
func (p *Peer) NewClient(server string) (*pb.GossipClient, error) {
	var (
		conn   *grpc.ClientConn
		err    error
		client pb.EchoClient
	)
	conn, err = grpc.Dial(server, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	client = pb.NewGossipClient(conn)
	return client, nil
}

func (p *Peer) randomView() []string {
}

func (p *Peer) addToBuffer(u *pb.Update) {
	var sm = &StoredMessage{
		Payload: u.Payload,
		SendCount: 0,
	}
	p.Buffer = append(b.Buffer, sm)
}

// newUUID generates a random UUID according to RFC 4122
func newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}
*/