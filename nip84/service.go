package nip84

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/dextryz/tenet"

	"github.com/dextryz/tenet/slicedb"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

type Service struct {
	Log *slog.Logger
	db  *slicedb.EventStore
	cfg *tenet.Config
}

func New(l *slog.Logger, d *slicedb.EventStore, c *tenet.Config) Service {
	return Service{
		Log: l,
		db:  d,
		cfg: c,
	}
}

func (s Service) RequestByNevent(ctx context.Context, nevent string) (*tenet.Highlight, error) {

	// TODO: udpate to nevent nip-19
	filter := nostr.Filter{
		IDs:   []string{nevent},
		Kinds: []int{tenet.KindHighlight},
		Limit: 1,
	}

	//events := s.queryRelays(ctx, filter)

	// 3. Convert the nostr events to current domain language (Highlights)

	//s.Log.Info("highlights found", "count", len(events))

	ch, err := s.db.QueryEvents(ctx, filter)
	if err != nil {
		return nil, err
	}

	s.Log.Info("highlights found", "count", len(ch))

	h := []*tenet.Highlight{}
	for e := range ch {
		a, err := tenet.ParseHighlight(*e)
		if err != nil {
			return nil, err
		}
		h = append(h, &a)
	}

	return h[0], nil
}

func (s Service) RequestByNaddr(ctx context.Context, naddr string) (tenet.HighlightMap, error) {

	// 1. Create the REQ filters for relays.

	prefix, data, err := nip19.Decode(naddr)
	if err != nil {
		if err.Error() == "incomplete naddr" {
			ep := data.(nostr.EntityPointer)
			if ep.Kind == 0 {
				fmt.Println("CCCC")
			}
			if ep.Identifier == "" {
				fmt.Println("AAAA")
				return nil, tenet.ErrEmptyIdentifier
			}
			if ep.PublicKey == "" {
				fmt.Println("BBBB")
				return nil, tenet.ErrEmptyPubKey
			}
		} else {
			return nil, err
		}
	}
	if prefix != "naddr" {
		return nil, fmt.Errorf("not a naddr URI: %s", naddr)
	}
	ep := data.(nostr.EntityPointer)

	tag := fmt.Sprintf("%d:%s:%s", ep.Kind, ep.PublicKey, ep.Identifier)

	f := nostr.Filter{
		Kinds: []int{tenet.KindHighlight},
		Tags: nostr.TagMap{
			"a": []string{tag},
		},
		Limit: 500,
	}

	// 2. Query the relays for events using filter

	events := s.queryRelays(ctx, f)

	// 3. Convert the nostr events to current domain language (Highlights)

	h := make(tenet.HighlightMap)
	for _, e := range events {

		// Cache event
		err := s.db.SaveEvent(ctx, e)
		if err != nil {
			return nil, err
		}

		a, err := tenet.ParseHighlight(*e)
		if err != nil {
			return nil, err
		}

		h[e.PubKey] = append(h[e.PubKey], &a)
	}

	return h, nil
}

// 1. Pull article highlights from cache
// 2. Update article content with highlights
func (s Service) ApplyToContent(ctx context.Context, a *tenet.Article) error {

	tag := fmt.Sprintf("%d:%s:%s", nostr.KindArticle, a.PubKey, a.Identifier)

	s.Log.Info("applying highlights to article", "a-tag", tag)

	filter := nostr.Filter{
		Kinds: []int{tenet.KindHighlight},
		Tags: nostr.TagMap{
			"a": []string{tag},
		},
		Limit: 500,
	}

	//events := s.queryRelays(ctx, filter)

	// 3. Convert the nostr events to current domain language (Highlights)

	//s.Log.Info("highlights found", "count", len(events))

	ch, err := s.db.QueryEvents(ctx, filter)
	if err != nil {
		return err
	}

	s.Log.Info("highlights found", "count", len(ch))

	for e := range ch {

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
