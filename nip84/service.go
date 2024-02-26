package nip84

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/dextryz/tenet"

	nos "github.com/dextryz/nostr"
	"github.com/dextryz/tenet/db"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

type Service struct {
	Log *slog.Logger
	db  *db.EventStore
	cfg *nos.Config
}

func New(l *slog.Logger, d *db.EventStore, c *nos.Config) Service {
	return Service{
		Log: l,
		db:  d,
		cfg: c,
	}
}

func (s Service) Request(ctx context.Context, naddr string) ([]*tenet.Highlight, error) {

	// 1. Create the REQ filters for relays.

	prefix, data, err := nip19.Decode(naddr)
	if err != nil {
		return nil, err
	}
	if prefix != "naddr" {
		return nil, fmt.Errorf("not a naddr URI: %s", naddr)
	}
	ep := data.(nostr.EntityPointer)

	tag := fmt.Sprintf("%d:%s:%s", ep.Kind, ep.PublicKey, ep.Identifier)

	f := nostr.Filter{
		Kinds: []int{nos.KindHighlight},
		Tags: nostr.TagMap{
			"a": []string{tag},
		},
		Limit: 500,
	}

	// 2. Query the relays for events using filter

	events := s.queryRelays(ctx, f)

	// 3. Convert the nostr events to current domain language (Highlights)

	h := []*tenet.Highlight{}
	for _, e := range events {

		//         // Cache event
		//         err := s.db.SaveEvent(ctx, e)
		// 		if err != nil {
		// 			return nil, err
		// 		}

		a, err := tenet.ParseHighlight(*e)
		if err != nil {
			return nil, err
		}
		h = append(h, &a)
	}

	return h, nil
}

// 1. Pull article highlights from cache
// 2. Update article content with highlights
func (s Service) ApplyToContent(ctx context.Context, a *tenet.Article) error {

	tag := fmt.Sprintf("%d:%s:%s", nostr.KindArticle, a.PubKey, a.Identifier)

	s.Log.Info("applying highlights to article", "a-tag", tag)

	filter := nostr.Filter{
		Kinds: []int{nos.KindHighlight},
		Tags: nostr.TagMap{
			"a": []string{tag},
		},
		Limit: 500,
	}

	events := s.queryRelays(ctx, filter)

	// 3. Convert the nostr events to current domain language (Highlights)

	s.Log.Info("highlights found", "count", len(events))

	for _, e := range events {

		a.Content = strings.ReplaceAll(
			a.Content,
			e.Content,
			fmt.Sprintf("<span class='highlight'>%s</span>", e.Content),
		)

		// 		if strings.Contains(a.Content, e.Content) {
		// 			a.Content = strings.ReplaceAll(
		// 				a.Content,
		// 				e.Content,
		// 				fmt.Sprintf("<span class='highlight'>%s</span>", e.Content),
		// 			)
		// 		}
	}

	// 	ch, err := s.db.QueryEvents(ctx, filter)
	// 	if err != nil {
	// 		return err
	// 	}
	//
	// 	s.Log.Info("highlights found", "count", len(ch))
	//
	// 	for e := range ch {
	// 		if strings.Contains(a.Content, e.Content) {
	// 			a.Content = strings.ReplaceAll(
	// 				a.Content,
	// 				e.Content,
	// 				fmt.Sprintf("<span class='highlight'>%s</span>", e.Content),
	// 			)
	// 		}
	// 	}

	return nil
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
