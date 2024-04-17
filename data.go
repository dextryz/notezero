package notezero

import (
	"context"
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr/nip19"
)

type TemplateID int

const (
	Profile TemplateID = iota
	ListArticle
	ListHighlight
	Article
	Highlight
	Unkown
)

// Has a parent event and a set of children:
// 1. Parent is 30023 - Children is 9802
// 2. Parent is 0 - Children is 30023
type Data struct {
	TemplateId TemplateID
	Event      EnhancedEvent
	Metadata   ProfileMetadata // Obviously always needs the author profile data
	Notes      []EnhancedEvent // For example, if the parent is an article, the children will be highlights
	Npub       string
	Naddr      string
	NaddrNaked string
	CreatedAt  string
	ModifiedAt string
	Kind       string
	Content    string
}

// FIXME: Remove the content bool hack
func (s *Handler) requestData(ctx context.Context, code string, content bool) (*Data, error) {

	if code == "" {

		// 1. Request parent event
		data := Data{
			TemplateId: ListArticle,
		}

		events, err := s.service.RequestEventFromCuratedAuthors(ctx, code)
		if err != nil {
			return nil, err
		}

		for _, e := range events {
			data.Notes = append(data.Notes, EnhancedEvent{Event: e})
		}

		return &data, nil
	}

	// 1. Request parent event
	rootEvent, err := s.service.RequestEvent(ctx, code)
	if err != nil {
		return nil, err
	}

	data := &Data{
		Event: EnhancedEvent{
			Event:  rootEvent,
			Relays: []string{},
		},
	}

	npub, _ := nip19.EncodePublicKey(rootEvent.PubKey)
	data.CreatedAt = rootEvent.CreatedAt.Time().String()
	data.Npub = npub // hopefully will be replaced later
	data.Naddr = ""
	data.NaddrNaked = ""
	data.CreatedAt = time.Unix(int64(rootEvent.CreatedAt), 0).Format("2006-01-02 15:04:05")
	data.ModifiedAt = time.Unix(int64(rootEvent.CreatedAt), 0).Format("2006-01-02T15:04:05Z07:00")

	if rootEvent.Kind >= 30000 && rootEvent.Kind < 40000 {
		if d := rootEvent.Tags.GetFirst([]string{"d", ""}); d != nil {
			//data.Naddr, _ = nip19.EncodeEntity(rootEvent.PubKey, rootEvent.Kind, d.Value(), relaysForNip19)
			data.NaddrNaked, _ = nip19.EncodeEntity(rootEvent.PubKey, rootEvent.Kind, d.Value(), nil)
		}
	}

	// 2. If not NPUB, then get the author profile

	// 3. Populate the children
	switch rootEvent.Kind {
	case 0:
		data.TemplateId = ListArticle
		events, err := s.service.AuthorArticles(ctx, npub)
		if err != nil {
			return nil, err
		}
		s.log.Info("articles pulled as children", "count", len(events))
		// TODO: Populate data.Notes with the list of requested articles.
		// This will be rendered using the ListArticle template.
		for _, e := range events {
			data.Notes = append(data.Notes, EnhancedEvent{Event: e})
		}
	case 30023:

		data.TemplateId = Article
		data.Content = mdToHtml(rootEvent.Content)

		if content {

			// 1. Process a list of kind 9082
			// 2. Use the first identifier of the article to request highlight data
			// 3. Add the highlights to the data.Notes list
			if d := rootEvent.Tags.GetFirst([]string{"d", ""}); d != nil {

				events, err := s.service.ArticleHighlights(ctx, rootEvent.Kind, rootEvent.PubKey, d.Value())
				if err != nil {
					return nil, err
				}

				// Add highlight notes to article data structure after applying to content
				for _, v := range events {
					data.Notes = append(data.Notes, EnhancedEvent{Event: v})
				}

				highlights := []string{}
				for _, v := range events {
					highlights = append(highlights, v.Content)
					fmt.Println(v.Content)
				}

				intervals := highlightIntervals(data.Content, highlights)
				merged := mergeIntervals(intervals)
				data.Content = highlight(data.Content, merged)
			}
		}

	default:
		data.TemplateId = Unkown
	}

	return data, nil
}
