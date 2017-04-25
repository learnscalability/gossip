// This construct represents the peer's HTTP REST command interface
package main

import (
	"encoding/json"
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
	return &cs
}

func (cs *CmdServer) Run() {
	log.Printf("HTTP command server started on %s\n", cs.server.Addr)
	cs.server.ListenAndServe()
}

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
	err = cs.peer.Send(&sp)
	if err != nil {
		log.Printf("Failed to send message to peer %+v with error %+v", sp, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
