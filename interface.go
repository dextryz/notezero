package notezero

import (
	"context"

	"github.com/nbd-wtf/go-nostr"
)

type EventService interface {
	RequestEventFromCuratedAuthors(ctx context.Context, code string) ([]*nostr.Event, error)
	RequestEvent(ctx context.Context, code string) (*nostr.Event, error)
	AuthorArticles(ctx context.Context, npub string) ([]*nostr.Event, error)
	ArticleHighlights(ctx context.Context, kind int, pubkey, identifier string) ([]*nostr.Event, error)
}
