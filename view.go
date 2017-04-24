package main

import (
	"fmt"
	"net"
)

// RemotePeer is a peer in the current peer's world view.
type RemotePeer struct {
	Pid string
	// Address were the remote peer listens for datagrams.
	raddr *net.UDPAddr
	// Connection to the remote udp peer.
	conn *net.UDPConn
}

// View is a set of remote peers indexed by their id.
type View struct {
	// peers is a lis of all remote peers.
	peers map[string]*RemotePeer
	// laddr is the address of the local peer which is connecting to all the remote peers.
	laddr *net.UDPAddr
}

// NewView create a new set of remote peers and build connections to all of them.
func NewView(laddr *net.UDPAddr, configs []PeerConfig) (*View, error) {
	var (
		v   View
		pc  PeerConfig
		err error
	)
	v = View{
		peers: make(map[string]*RemotePeer),
		laddr: laddr,
	}
	for _, pc = range configs {
		err = v.AddPeer(pc)
		if err != nil {
			return nil, err
		}
	}
	return &v, nil
}

func (v *View) Get(pid string) (*RemotePeer, error) {
	var (
		rp    *RemotePeer
		found bool
	)
	if rp, found = v.peers[pid]; found {
		return rp, nil
	} else {
		return nil, fmt.Errorf("Remote Peer with pid %s not registered", pid)
	}
}

// AddPeer registers a new RemotePeer to the View and stores a connection to it.
func (v *View) AddPeer(pc PeerConfig) error {
	var (
		exists bool
		rp     *RemotePeer
		raddr  *net.UDPAddr
		conn   *net.UDPConn
		err    error
	)
	if _, exists = v.peers[pc.Pid]; exists {
		return nil
	}
	raddr, err = net.ResolveUDPAddr("udp", pc.Bind)
	if err != nil {
		return fmt.Errorf("Failed to prepare peer address to send datagrams to: %+v", err)
	}
	fmt.Println(">>>>>>>>>>>>>>", v.laddr, raddr)
	conn, err = net.DialUDP("udp", v.laddr, raddr)
	if err != nil {
		return fmt.Errorf("Failed to connect to remote peer UDP server %+v with error: %+v\n", pc, err)
	}
	rp = &RemotePeer{
		Pid:   pc.Pid,
		raddr: raddr,
		conn:  conn,
	}
	v.peers[rp.Pid] = rp
	return nil
}

// RemovePeer closes the UDP connection and remote the remote peer from the list.
func (v View) RemovePeer(pid string) error {
	var (
		rp = v.peers[pid]
	)
	delete(v.peers, pid)
	return rp.conn.Close()
}

// Cleanup terminates all connections to the remote peers.
func (v View) Cleanup() error {
	var (
		rp  *RemotePeer
		err error
	)
	for _, rp = range v.peers {
		err = v.RemovePeer(rp.Pid)
		if err != nil {
			return err
		}
	}
	return nil
}
