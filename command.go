package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

const (
	updatePeerType = yota
	udpateViewType
	updateRandomType
	joinType
)

type updatePeerCommand struct {
	Id string `json:"name"`
	Content string `json:"content"`
}

type updateViewCommand struct {
	Content string `json:"content"`
}

type updateRandomCommand struct {
	Percentage int `json:"percentage"`
	Num int `json:"num"`
	Content string `json:"content"`
}

type joinCommand struct {
	Id string `json:"id"`
	Bind string `json:"bind"`
}

type commandServer struct {
	server *http.Server
	commands chan<- interface{}
}

func newCommandServer(bind string, commands chan<- interface{}) *commandServer {
	var (
		mux    *http.ServeMux
		server *http.Server
		cs     CommandServer
	)
	mux = http.NewServeMux()
	server = &http.Server{
		Handler: mux,
		Addr:    bind,
	}
	cs = CommandServer{server, commands}
	mux.HandleFunc("/update/peer", cs.commandHandler(updatePeerType))
	mux.HandleFunc("/update/view", cs.commandHandler(updateViewType))
	mux.HandleFunc("/update/random", cs.commandHandler(updateRandomType))
	mux.HandleFunc("/join", cs.commandHandler(joinType))
	return &cs
}

func (cs *commandServer) listen() error {
	log.Printf("HTTP command server started on %s\n", cs.server.Addr)
	return cs.server.ListenAndServe()
}

func (cs *commandServer) terminate() error {
	log.Println("HTTP command server terminating")
	return cs.server.Close()
}

type httpHandler func(http.ResponseWriter, *http.Request)

func (cs *commandServer) commandHandler(typ int) httpHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			decoder = json.NewDecoder(r.Body)
			err error
			cmd interface{}
		)
		switch typ {
		case updatePeerType:
			cmd = updatePeerCommand{}
			err = decoder.Decode(&cmd)
		case updateViewType:
			cmd = updateViewCommand{}
			err = decoder.Decode(&cmd)
		case updateRandomType:
			cmd = updateRandomCommand{}
			err = decoder.Decode(&cmd)
		case joinType:
			cmd = joinCommand{}
			err = decoder.Decode(&cmd)
		default:
			err = fmt.Errorf("unrecognized payload command type: %d", typ)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		select {
		case cs.commands <- &cmd:
		case r.Context().Done():
			http.Error(w, "server closing. retry!", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
