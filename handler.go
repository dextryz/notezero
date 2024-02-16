package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"text/template"

	"github.com/dextryz/nip84"
	nos "github.com/dextryz/nostr"

	"github.com/fiatjaf/eventstore"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

var ErrNotFound = errors.New("todo list not found")

type EventStore struct {
	eventstore.Store
	UpdatedAt map[string]nostr.Timestamp
}

type Handler struct {
	cfg *nos.Config
	db  *EventStore
}

func (s *Handler) Close() error {
	return nil
}

func (s *Handler) Home(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("static/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.ExecuteTemplate(w, "index.html", "")
	if err != nil {
		fmt.Println("Error executing template:", err)
	}
}

func (s *Handler) Highlights(w http.ResponseWriter, r *http.Request) {

	naddr := r.URL.Query().Get("naddr")

	log.Printf("pulling article hightlights for %s", naddr)

	ctx := context.Background()
	events := s.queryHighlights(ctx, naddr)

	log.Printf("%d highlights pulled", len(events))

	h := []*nip84.Highlight{}
	for _, e := range events {
		a, err := nip84.ToHighlight(e)
		if err != nil {
			log.Fatalln(err)
		}
		h = append(h, &a)
	}

	log.Println(h[0])

	tmpl, err := template.ParseFiles("static/card.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, h)
}

func (s *Handler) Validate(w http.ResponseWriter, r *http.Request) {

	pk := r.URL.Query().Get("search")

	if pk != "" {

		prefix, _, err := nip19.Decode(pk)

		if err != nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<span class="message error">Invalid entity</span>`))
			return
		}

		if prefix != "naddr" {
			log.Println("start with naddr")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<span class="message error">Start with npub</span>`))
			return
		}

		// Add text to show valid if you want to.
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<span class="message success"> </span>`))
	}
}
