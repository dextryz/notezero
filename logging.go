package notezero

import (
	"context"
	"log/slog"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

type logging struct {
	log  *slog.Logger
	next EventService
}

func NewLogging(log *slog.Logger, next EventService) logging {
	return logging{
		log:  log,
		next: next,
	}
}

func (s logging) Profile(ctx context.Context, npub string) (*nostr.Event, error) {

	s.log.Info("requesting author profile", "service", "EventService")

	defer func(start time.Time) {
		s.log.Info(
			"Profile",
			"npub", npub,
			"took", time.Since(start),
		)
	}(time.Now())

	return s.next.Profile(ctx, npub)
}

// 1. Check if the event is in the cache
// 2. If not, request event from the set of relays
func (s logging) RequestEvent(ctx context.Context, code string) (evt *nostr.Event, err error) {

	s.log.Info("requesting events", "service", "EventService")

	defer func(start time.Time) {
		s.log.Info(
			"RequestEvent",
			"code", code,
			"err", err,
			"took", time.Since(start),
		)
	}(time.Now())

	return s.next.RequestEvent(ctx, code)

}

func (s logging) AuthorArticles(ctx context.Context, npub string) ([]*nostr.Event, error) {
	s.log.Info("event retrieved from relays", "npub", npub)
	return s.next.AuthorArticles(ctx, npub)
}

func (s logging) ArticleHighlights(ctx context.Context, kind int, pubkey, identifier string) ([]*nostr.Event, error) {
	return s.next.ArticleHighlights(ctx, kind, pubkey, identifier)
}
