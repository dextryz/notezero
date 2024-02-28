package nip23

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/dextryz/tenet"

	"github.com/dextryz/tenet/slicedb"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

type Service struct {
	Log *slog.Logger
	Db  *slicedb.EventStore
	cfg *tenet.Config
}

func New(l *slog.Logger, d *slicedb.EventStore, c *tenet.Config) Service {
	return Service{
		Log: l,
		Db:  d,
		cfg: c,
	}
}

func (s Service) Request(ctx context.Context, naddr string) (tenet.Article, error) {

	a := tenet.Article{}

	prefix, data, err := nip19.Decode(naddr)
	if err != nil {
		return a, err
	}
	if prefix != "naddr" {
		return a, fmt.Errorf("incorrect prefix: %s", naddr)
	}
	ep := data.(nostr.EntityPointer)
	if ep.Kind != nostr.KindArticle {
		return a, err
	}

	s.Log.Info("requesting article from relays", "identifier", ep.Identifier, "pubkey", ep.PublicKey, "kind", ep.Kind)

	filter := nostr.Filter{
		Authors: []string{ep.PublicKey},
		Kinds:   []int{ep.Kind},
		Tags: nostr.TagMap{
			"d": []string{ep.Identifier},
		},
	}

	events := s.queryRelays(ctx, filter)

	// 	pool := nostr.NewSimplePool(ctx)
	//     s.Log.Info("querying relays", "count", len(s.cfg.Relays))
	// 	e := pool.QuerySingle(ctx, s.cfg.Relays, filter)

	s.Log.Info("events received from relays", "count", len(events))

	a, err = tenet.ParseArticle(*events[0])
	if err != nil {
		return a, err
	}

	a, err = MdToHtml(&a)
	if err != nil {
		return a, err
	}

	return a, nil
}

func (s *Service) queryRelays(ctx context.Context, filter nostr.Filter) (ev []*nostr.Event) {

	var m sync.Map
	var wg sync.WaitGroup
	for _, url := range s.cfg.Relays {

		wg.Add(1)
		go func(wg *sync.WaitGroup, url string) {
			defer wg.Done()

			r, err := nostr.RelayConnect(ctx, url)
			if err != nil {
				panic(err)
			}

			events, err := r.QuerySync(ctx, filter)
			if err != nil {
				s.Log.Error("failed to query events", slog.Any("error", err))
			}

			for _, e := range events {
				m.Store(e.ID, e)
			}

		}(&wg, url)
	}
	wg.Wait()

	m.Range(func(_, v any) bool {
		ev = append(ev, v.(*nostr.Event))
		return true
	})

	return ev
}
