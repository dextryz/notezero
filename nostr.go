package notezero

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/dextryz/notezero/badger"
	"github.com/fiatjaf/eventstore"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

type Nostr struct {
	db     eventstore.Store
	cache  *badger.Cache
	relays []string
}

func NewNostr(db eventstore.Store, cache *badger.Cache, relays []string) Nostr {
	return Nostr{
		db:     db,
		cache:  cache,
		relays: relays,
	}
}

func (s Nostr) pullProfileList(ctx context.Context, npubs []string) ([]*nostr.Event, error) {

	filter := nostr.Filter{
		Kinds: []int{nostr.KindProfileMetadata},
		Limit: 500,
	}

	for _, npub := range npubs {
		_, pk, err := nip19.Decode(npub)
		if err != nil {
			return nil, err
		}
		filter.Authors = append(filter.Authors, pk.(string))
	}

	pool := nostr.NewSimplePool(context.Background())

	latestNotes := func() <-chan *nostr.Event {
		notes := make(chan *nostr.Event)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			defer fmt.Println("latestNotes producer exited")
			defer close(notes)
			ch := pool.SubManyEose(ctx, s.relays, nostr.Filters{filter})
			for {
				select {
				case ie, more := <-ch:
					if !more {
						return
					}
					notes <- ie.Event
					s.db.SaveEvent(ctx, ie.Event)
				case <-ctx.Done():
					return
				}
			}
		}()
		return notes
	}

	profiles, err := eventstore.RelayWrapper{Store: s.db}.QuerySync(ctx, filter)
	if err != nil {
		return nil, err
	}

	if len(profiles) < len(CURATED_LIST) {
		for n := range latestNotes() {
			profiles = append(profiles, n)
		}
	}

	return profiles, nil
}

func (s Nostr) pullNextArticlePage(ctx context.Context, npubs []string) ([]*nostr.Event, error) {

	filter := nostr.Filter{
		Kinds: []int{nostr.KindArticle},
		Until: &pageUntil,
		Limit: pageLimit,
	}

	for _, npub := range npubs {
		_, pk, err := nip19.Decode(npub)
		if err != nil {
			return nil, err
		}
		filter.Authors = append(filter.Authors, pk.(string))
	}

	pool := nostr.NewSimplePool(context.Background())

	latestNotes := func() <-chan *nostr.Event {
		notes := make(chan *nostr.Event)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			defer fmt.Println("latestNotes producer exited")
			defer close(notes)
			ch := pool.SubManyEose(ctx, s.relays, nostr.Filters{filter})
			for {
				select {
				case ie, more := <-ch:
					if !more {
						return
					}
					notes <- ie.Event
					s.db.SaveEvent(ctx, ie.Event)
				case <-ctx.Done():
					return
				}
			}
		}()
		return notes
	}

	// Fetch from local store if available
	lastNotes, err := eventstore.RelayWrapper{Store: s.db}.QuerySync(ctx, filter)
	if err != nil {
		return nil, err
	}

	if len(lastNotes) < pageLimit {
		for n := range latestNotes() {
			lastNotes = append(lastNotes, n)
		}
	}

	slices.SortFunc(lastNotes, func(a, b *nostr.Event) int { return int(b.CreatedAt - a.CreatedAt) })

	pageUntil = lastNotes[len(lastNotes)-1].CreatedAt - 1

	//	for _, v := range lastNotes {
	//		fmt.Printf("pageUntil: %v\n", time.Unix(int64(v.CreatedAt), 0).Format("2006-01-02 15:04:05"))
	//	}

	//	fmt.Printf("last pageUntil: %v\n", time.Unix(int64(pageUntil), 0).Format("2006-01-02 15:04:05"))

	return lastNotes, nil
}
