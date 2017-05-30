package main

// View is a set of remote peers indexed by their id.
type View map[string]PeerConfig

// NewView create a new set of remote peers and build connections to all of them.
func NewView(configs []PeerConfig) View {
	var (
		v  View
		pc PeerConfig
	)
	v = make(View)
	for _, pc = range configs {
		v[pc.Pid] = pc
	}
	return v
}

// AddPeer registers a new RemotePeer to the View and stores a connection to it.
func (v View) AddPeer(pc PeerConfig) bool {
	var exists bool
	if _, exists = v[pc.Pid]; !exists {
		v[pc.Pid] = pc
	}
	return exists
}

// RemovePeer closes the UDP connection and remote the remote peer from the list.
func (v View) RemovePeer(pid string) {
	delete(v, pid)
}
