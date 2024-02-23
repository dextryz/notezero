package db

import (
	"github.com/fiatjaf/eventstore"
	"github.com/fiatjaf/eventstore/slicestore"
	"github.com/nbd-wtf/go-nostr"
)

type EventStore struct {
	eventstore.Store
	UpdatedAt map[string]nostr.Timestamp
}

func New() (*EventStore, error) {

	db := &slicestore.SliceStore{}

	err := db.Init()
	if err != nil {
		return nil, err
	}

	return &EventStore{
		Store:     db,
		UpdatedAt: make(map[string]nostr.Timestamp),
	}, nil
}
