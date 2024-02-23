package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	nos "github.com/dextryz/nostr"
	"github.com/dextryz/tenet/db"
	"github.com/dextryz/tenet/handler"
	"github.com/dextryz/tenet/service"
)

func main() {

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	store, err := db.New()
	if err != nil {
		log.Error("failed to create store", slog.Any("error", err))
		os.Exit(1)
	}

	cfg, err := nos.LoadConfig(os.Getenv("NOSTR"))
	if err != nil {
		panic(err)
	}

	s := service.New(log, store, cfg)
	h := handler.New(log, s)

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	mux.HandleFunc("/", h.View)
	mux.HandleFunc("GET /highlights", h.Get)

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
