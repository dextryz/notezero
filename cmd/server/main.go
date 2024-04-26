package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	nz "github.com/dextryz/notezero"

	"github.com/dextryz/notezero/badger"
	eventstore_badger "github.com/fiatjaf/eventstore/badger"
)

func main() {

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	log.Info("Starting")

	relays := []string{
		"wss://relay.damus.io/",
		"wss://nostr-01.yakihonne.com",
		"wss://relay.highlighter.com/",
		"wss://relay.f7z.io",
		"wss://nos.lol",
		"wss://nostr.wine/",
		"wss://purplepag.es/",
	}

	db := &eventstore_badger.BadgerBackend{
		Path: "eventstore.db",
	}
	err := db.Init()
	if err != nil {
		panic(err)
	}

	cache, err := badger.New(db.DB)
	if err != nil {
		log.Error("failed to create store", slog.Any("error", err))
		os.Exit(1)
	}

	const imgDir = "./static/img"
	err = os.MkdirAll(imgDir, 0777)
	if err != nil {
		slog.Error("unable to create dir", "err", err)
	}

	s := nz.NewEventService(db, cache, relays)
	l := nz.NewLogging(log, s)
	n := nz.NewNostr(db, imgDir, cache, relays)
	h := nz.NewHandler(log, l, n)
	defer n.Close()

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	mux.HandleFunc("/", h.CodeHandler)
	mux.HandleFunc("GET /search", h.RedirectSearch)
	mux.HandleFunc("GET /{code}", h.CodeHandler)
	mux.HandleFunc("GET /content/{naddr}", h.ContentHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         "0.0.0.0:" + port,
		Handler:      mux,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}

	fmt.Printf("Listening on %v\n", server.Addr)

	server.ListenAndServe()
}
