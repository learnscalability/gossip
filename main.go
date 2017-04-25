package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
)

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
