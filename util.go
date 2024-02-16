package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	nos "github.com/dextryz/nostr"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

func (s *Handler) queryRelays(ctx context.Context, filter nostr.Filter) (ev []*nostr.Event) {

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
				log.Fatalln(err)
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

func (s *Handler) queryHighlights(ctx context.Context, naddr string) []*nostr.Event {

	prefix, data, err := nip19.Decode(naddr)
	if err != nil {
		log.Fatalln(err)
	}
	if prefix != "naddr" {
		log.Fatalln(err)
	}
	ep := data.(nostr.EntityPointer)

	tag := fmt.Sprintf("%d:%s:%s", ep.Kind, ep.PublicKey, ep.Identifier)

	f := nostr.Filter{
		Kinds: []int{nos.KindHighlight},
		Tags: nostr.TagMap{
			"a": []string{tag},
		},
	}

	return s.queryRelays(ctx, f)
}
