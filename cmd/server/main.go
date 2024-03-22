package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/dextryz/notezero/badger"
	"github.com/dextryz/notezero/event"
	"github.com/dextryz/notezero/handler"
	eventstore_badger "github.com/fiatjaf/eventstore/badger"
)

func main() {

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	log.Info("Starting")

	relays := []string{
		"wss://relay.damus.io/",
		"wss://nostr-01.yakihonne.com",
		// "wss://nostr-02.yakihonne.com",
		// "wss://relay.highlighter.com/",
		"wss://relay.f7z.io",
		"wss://nos.lol",
	}

	db := &eventstore_badger.BadgerBackend{
		Path: "nostr.db",
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

	// Event service is responsible to communicating with relays and populating the cache.
	service := event.New(log, db, cache, relays)

	// Handle the templates and view model
	h := handler.New(log, service)

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	fs = http.FileServer(http.Dir("./fonts"))
	mux.Handle("/fonts/", http.StripPrefix("/fonts/", fs))

	fs = http.FileServer(http.Dir("./img"))
	mux.Handle("/img/", http.StripPrefix("/img/", fs))

	mux.HandleFunc("/", h.Homepage)
	mux.HandleFunc("GET /list", h.ListHandler)
	mux.HandleFunc("GET /articles/{naddr}", h.ArticleHandler)
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
