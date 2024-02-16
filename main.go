package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	nos "github.com/dextryz/nostr"
	"github.com/nbd-wtf/go-nostr"

	"github.com/fiatjaf/eventstore/slicestore"
)

var (
	ADDR = "127.0.0.1"
	PORT = "8080"
)

func main() {

	log.Println("starting server")

	path, ok := os.LookupEnv("NOSTR")
	if !ok {
		log.Fatalln("NOSTR env var not set")
	}

	cfg, err := nos.LoadConfig(path)
	if err != nil {
		panic(err)
	}

	db := &slicestore.SliceStore{}

	err = db.Init()
	if err != nil {
		panic(err)
	}

	es := EventStore{
		Store:     db,
		UpdatedAt: make(map[string]nostr.Timestamp),
	}

	h := Handler{
		cfg: cfg,
		db:  &es,
	}

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	mux.HandleFunc("/", h.Home)
	mux.HandleFunc("GET /validate", h.Validate)
	mux.HandleFunc("GET /highlights", h.Highlights)

	s := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", ADDR, PORT),
		Handler: mux,
	}

	// Create a channel to listen for OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		err := s.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	<-stop

	// Create a context with a timeout for the server's shutdown process
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = h.Close()
	if err != nil {
		log.Fatalf("closing subscriptions failed:%+v", err)
	}

	err = s.Shutdown(ctx)
	if err != nil {
		log.Fatalf("server shutdown failed:%+v", err)
	}

	log.Println("server gracefully stopped")
}
