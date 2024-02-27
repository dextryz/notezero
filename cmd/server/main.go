package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/dextryz/tenet"

	"github.com/dextryz/tenet/handler"
	"github.com/dextryz/tenet/nip01"
	"github.com/dextryz/tenet/nip23"
	"github.com/dextryz/tenet/nip84"
	"github.com/dextryz/tenet/slicedb"
	"github.com/dextryz/tenet/sqlite"
)

func main() {

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	log.Info("Starting")

	dbEvents, err := slicedb.New()
	if err != nil {
		log.Error("failed to create store", slog.Any("error", err))
		os.Exit(1)
	}

	// 	cfg, err := tenet.LoadConfig(os.Getenv("NOSTR"))
	// 	if err != nil {
	// 		panic(err)
	// 	}

	cfg := &tenet.Config{
		Relays: []string{
			"wss://relay.damus.io/",
			"wss://nostr-01.yakihonne.com",
			"wss://nostr-02.yakihonne.com",
			"wss://relay.highlighter.com/",
			"wss://relay.f7z.io",
			"wss://nos.lol",
		},
	}

	dbProfile := sqlite.New("profile.db")
	defer dbProfile.Close()

	ps := nip01.New(log, dbProfile, cfg)
	hs := nip84.New(log, dbEvents, cfg)
	as := nip23.New(log, dbEvents, cfg)

	h := handler.New(log, hs, ps, as)

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	mux.HandleFunc("/", h.View)
	mux.HandleFunc("GET /highlights", h.Highlights)
	mux.HandleFunc("GET /high/{nevent}", h.Highlight)
	mux.HandleFunc("GET /articles/{naddr}", h.Article)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         "127.0.0.1:" + port,
		Handler:      mux,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}

	fmt.Printf("Listening on %v\n", server.Addr)

	server.ListenAndServe()
}
