package nip01

import (
	"context"
	"fmt"
	"golang.org/x/exp/slog"
	"sync"

	"github.com/dextryz/tenet"
	"github.com/dextryz/tenet/sqlite"
	"github.com/nbd-wtf/go-nostr"
)

type Service struct {
	log *slog.Logger
	db  *sqlite.Db
	cfg *tenet.Config
}

func New(log *slog.Logger, db *sqlite.Db, cfg *tenet.Config) Service {
	return Service{
		log: log,
		db:  db,
		cfg: cfg,
	}
}

// Should nopt be able to edit profile
// 1. Request the sqlite cache for the profile
// 2. If a cache miss happens, REQ the relays.
// 3. After REQ, store in local cache
func (s Service) Request(ctx context.Context, pubkey string) (tenet.Profile, error) {

	profile := tenet.Profile{}

	// TODO: Impl cache check ( s.db.QueryProfile() )

	f := nostr.Filter{
		Kinds:   []int{nostr.KindProfileMetadata},
		Authors: []string{pubkey},
	}

	// Retrieve user profile from nostr relays
	metadata := s.queryRelays(ctx, f)
	if len(metadata) != 1 {
		fmt.Println(metadata)
		return profile, fmt.Errorf("cannot have more then one profile: %s", pubkey)
	}

	// Only one profile can be pulled per pubkey.
	p, err := tenet.ParseMetadata(*metadata[0])
	if err != nil {
		return profile, err
	}

	profile, err = s.db.StoreProfile(ctx, *p, pubkey)
	if err != nil {
		return profile, err
	}

	return profile, nil
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
				s.log.Error("failed to query events", slog.Any("error", err))
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
