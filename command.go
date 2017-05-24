// This construct represents the peer's HTTP REST command interface
package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type CmdServer struct {
	server *http.Server
	peer   *Peer
}

func NewCmdServer(peer *Peer) *CmdServer {
	var (
		mux    *http.ServeMux
		server *http.Server
		cs     CmdServer
	)
	mux = http.NewServeMux()
	server = &http.Server{
		Handler: mux,
		Addr:    peer.config.CmdBind,
	}
	cs = CmdServer{server, peer}
	mux.HandleFunc("/send", cs.sendHandler)
	mux.HandleFunc("/spread", cs.spreadHandler)
	mux.HandleFunc("/join", cs.joinHandler)
	return &cs
}

// Run starts the http listener. Should be run as a goroutine.
func (cs *CmdServer) Run() {
	log.Printf("HTTP command server started on %s\n", cs.server.Addr)
	cs.server.ListenAndServe()
}

// Close terminates the server and all connections without waiting for them
// to close gracefully.
func (cs *CmdServer) Close() {
	log.Println("HTTP command server terminating")
	cs.server.Close()
}

// SendPayload.
type SendPayload struct {
	Pid     string `json:"pid"`
	Content string `json:"content"`
}

// sendHandler expects a body of the following format: {pid: string, content: string}
func (cs *CmdServer) sendHandler(w http.ResponseWriter, r *http.Request) {
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
	err = cs.peer.Send(sp.Pid, sp.Content)
	if err != nil {
		log.Printf("Failed to send message to peer %+v with error %+v", sp, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (cs *CmdServer) spreadHandler(w http.ResponseWriter, r *http.Request) {
	var (
		payload []byte
		spayload string
		err error
		pid string
	)
	payload, err = ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("Failed to read request body with error: %+v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	spayload = string(payload)
	for pid = range cs.peer.view {
		err = cs.peer.Send(pid, spayload)
		if err != nil {
			log.Printf("Failed to send payload `%s` to peer id `%s` with error: %+v", spayload, pid, err)
		}
	}
}

type JoinPayload struct {

}

// TODO joinHandler
func (cs *CmdServer) joinHandler(w http.ResponseWriter, r *http.Request) {
}
