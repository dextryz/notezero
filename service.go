package notezero

import (
	"context"
	"fmt"
	"log"
	"slices"
	"sync"
	"time"

	"github.com/dextryz/notezero/badger"
	"github.com/fiatjaf/eventstore"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

var (
	pageLimit = 4
	pageUntil = nostr.Now()
)

type eventService struct {
	db     eventstore.Store
	cache  *badger.Cache
	relays []string
}

func NewEventService(db eventstore.Store, cache *badger.Cache, relays []string) eventService {
	return eventService{
		db:     db,
		cache:  cache,
		relays: relays,
	}
}

var CURATED_LIST = []string{
	"npub14ge829c4pvgx24c35qts3sv82wc2xwcmgng93tzp6d52k9de2xgqq0y4jk", // dextryz
	//"npub1m4ny6hjqzepn4rxknuq94c2gpqzr29ufkkw7ttcxyak7v43n6vvsajc2jl", // Laeserin
	//"npub1mu2tx4ue4yt7n7pymcql3agslnx0zeyt34zmmfex2g07k6ymtksq7hansc", // CYB3RX
	//	"npub18jvyjwpmm65g8v9azmlvu8knd5m7xlxau08y8vt75n53jtkpz2ys6mqqu3", // onigirl
	//	"npub180cvv07tjdrrgpa0j7j7tmnyl2yr6yr7l8j4s3evf6u64th6gkwsyjh6w6", // fiatjaf
	//	"npub1l2vyh47mk2p0qlsku7hg0vn29faehy9hy34ygaclpn66ukqp3afqutajft", // pablo
	//	"npub1r0rs5q2gk0e3dk3nlc7gnu378ec6cnlenqp8a3cjhyzu6f8k5sgs4sq9ac", // karnage
	//	"npub1mygerccwqpzyh9pvp6pv44rskv40zutkfs38t0hqhkvnwlhagp6s3psn5p", // gsoverienty
	//	"npub1xtscya34g58tk0z605fvr788k263gsu6cy9x0mhnm87echrgufzsevkk5s", // Will
	//	"npub18ams6ewn5aj2n3wt2qawzglx9mr4nzksxhvrdc4gzrecw7n5tvjqctp424", // Derek Ros
	//	"npub12262qa4uhw7u8gdwlgmntqtv7aye8vdcmvszkqwgs0zchel6mz7s6cgrkj", // semisol
	//	"npub1h8nk2346qezka5cpm8jjh3yl5j88pf4ly2ptu7s6uu55wcfqy0wq36rpev", // Dan Swann
	//	"npub1utx00neqgqln72j22kej3ux7803c2k986henvvha4thuwfkper4s7r50e8", // utxo
}

func (s eventService) Profile(ctx context.Context, npub string) (*nostr.Event, error) {

	_, pk, err := nip19.Decode(npub)
	if err != nil {
		return nil, err
	}

	filter := nostr.Filter{
		Kinds:   []int{nostr.KindProfileMetadata},
		Authors: []string{pk.(string)},
	}

	wdb := eventstore.RelayWrapper{Store: s.db}

	// Try to fetch in our internal eventstore (cache) first
	events, err := wdb.QuerySync(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(events) != 0 {
		return events[0], nil
	}

	// No events found in cache, request relays and publish to cache
	events = s.queryRelays(ctx, filter)
	for _, e := range events {
		err := wdb.Publish(ctx, *e)
		if err != nil {
			return nil, err
		}
	}

	return events[0], nil
}

func (s eventService) RequestEvent(ctx context.Context, code string) (*nostr.Event, error) {

	// Create a nostr filter from the NIP-19 code
	prefix, data, err := nip19.Decode(code)
	if err != nil {
		return nil, err
	}

	var filter nostr.Filter

	switch v := data.(type) {
	case nostr.EntityPointer:
		filter.Authors = []string{v.PublicKey}
		filter.Tags = nostr.TagMap{
			"d": []string{v.Identifier},
		}
		if v.Kind != 0 {
			filter.Kinds = append(filter.Kinds, v.Kind)
		}
	case string:
		if prefix == "npub" {
			filter.Authors = []string{v}
			filter.Kinds = []int{0}
		}
	default:
		return nil, fmt.Errorf("code type not supported: %s", code)
	}

	// Wrap the cache db to be used with a relay interface
	wdb := eventstore.RelayWrapper{Store: s.db}

	// Try to fetch in our internal eventstore (cache) first
	events, err := wdb.QuerySync(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(events) != 0 {
		return events[0], nil
	}

	// No events found in cache, request relays and publish to cache
	events = s.queryRelays(ctx, filter)
	for _, e := range events {
		err := wdb.Publish(ctx, *e)
		if err != nil {
			return nil, err
		}
	}

	return events[0], nil
}

func (s eventService) PullLatest(ctx context.Context, npubs []string) ([]*nostr.Event, error) {

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

	var lastNotes []*nostr.Event
	// fetch from local store if available
	//lastNotes, _ = eventstore.RelayWrapper{Store: s.db}.QuerySync(ctx, filter)
	fmt.Println("START READING")
	for n := range latestNotes() {
		lastNotes = append(lastNotes, n)
	}
	fmt.Println("DONE READING")

	//if len(lastNotes) < 5 {

	//		consumer := latestNotes()
	//		for i := 0; i < 10; i++ {
	//			n := <-consumer
	//			lastNotes = append(lastNotes, n)
	//		}
	//}

	slices.SortFunc(lastNotes, func(a, b *nostr.Event) int { return int(b.CreatedAt - a.CreatedAt) })

	for _, v := range lastNotes {
		fmt.Printf("pageUntil: %v\n", time.Unix(int64(v.CreatedAt), 0).Format("2006-01-02 15:04:05"))
	}

	pageUntil = lastNotes[len(lastNotes)-1].CreatedAt - 1
	fmt.Printf("last pageUntil: %v\n", time.Unix(int64(pageUntil), 0).Format("2006-01-02 15:04:05"))

	return lastNotes, nil
}

func (s eventService) AuthorArticles(ctx context.Context, npub string) ([]*nostr.Event, error) {

	_, pk, err := nip19.Decode(npub)
	if err != nil {
		return nil, err
	}

	filter := nostr.Filter{
		Kinds:   []int{nostr.KindArticle},
		Authors: []string{pk.(string)},
		Limit:   pageLimit,
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

	var lastNotes []*nostr.Event
	// fetch from local store if available
	//lastNotes, _ = eventstore.RelayWrapper{Store: s.db}.QuerySync(ctx, filter)
	fmt.Println("START READING")
	for n := range latestNotes() {
		lastNotes = append(lastNotes, n)
	}
	fmt.Println("DONE READING")

	//if len(lastNotes) < 5 {

	//		consumer := latestNotes()
	//		for i := 0; i < 10; i++ {
	//			n := <-consumer
	//			lastNotes = append(lastNotes, n)
	//		}
	//}

	slices.SortFunc(lastNotes, func(a, b *nostr.Event) int { return int(b.CreatedAt - a.CreatedAt) })

	return lastNotes, nil
}

func (s eventService) ArticleHighlights(ctx context.Context, kind int, pubkey, identifier string) ([]*nostr.Event, error) {

	wdb := eventstore.RelayWrapper{Store: s.db}

	pool := nostr.NewSimplePool(context.Background())

	// 2. Article is cached, so pull highlights

	tag := fmt.Sprintf("%d:%s:%s", kind, pubkey, identifier)

	filter := nostr.Filter{
		Kinds: []int{9802},
		Tags: nostr.TagMap{
			"a": []string{tag},
		},
		Limit: 500,
	}
	var lastNotes []*nostr.Event

	// fetch from external relays asynchronously
	external := make(chan []*nostr.Event)
	go func() {
		notes := make([]*nostr.Event, 0, filter.Limit)
		defer func() {
			external <- notes
		}()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		ch := pool.SubManyEose(ctx, s.relays, nostr.Filters{filter})
		for {
			select {
			case ie, more := <-ch:
				if !more {
					return
				}
				notes = append(notes, ie.Event)
				s.db.SaveEvent(ctx, ie.Event)
				s.cache.Set(identifier, []byte{})
			case <-ctx.Done():
				return
			}
		}
	}()

	// fetch from local store if available
	if _, found := s.cache.Get(identifier); found {
		lastNotes, _ = wdb.QuerySync(ctx, filter)
	} else {
		// if we didn't get enough notes (or if we didn't even query the local store), wait for the external relays
		lastNotes = <-external
		s.cache.Set(identifier, []byte{})

		// 		tags := nostr.Tags{
		// 			{"a", tag},
		// 		}
		//
		// 		e := nostr.Event{
		// 			Kind:      9802,
		// 			PubKey:    pubkey,
		// 			Content:   "",
		// 			CreatedAt: nostr.Now(),
		// 			Tags:      tags,
		// 		}
		//
		// 		// USe the server secret key, makes it easy to filer using pubkey
		// 		sk := os.Getenv("NOSTR_SK")
		// 		_ = e.Sign(sk)
		//
		// 		s.db.SaveEvent(ctx, &e)
		//
		// 		// Add a dummy
		// 		lastNotes = append(lastNotes, &e)
	}

	return lastNotes, nil
}

func (s *eventService) queryRelays(ctx context.Context, filter nostr.Filter) (ev []*nostr.Event) {

	var m sync.Map
	var wg sync.WaitGroup
	for _, url := range s.relays {

		wg.Add(1)
		go func(wg *sync.WaitGroup, url string) {
			defer wg.Done()

			r, err := nostr.RelayConnect(ctx, url)
			if err != nil {
				log.Fatalf("panicing to query relays: %v", err)
				panic(err)
			}

			events, err := r.QuerySync(ctx, filter)
			if err != nil {
				// TODO
				return
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
