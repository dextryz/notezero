package handler

import (
	"context"
	"time"

	"github.com/dextryz/tenet"
	"github.com/nbd-wtf/go-nostr/nip19"
)

func (s *Handler) requestData(ctx context.Context, code string) (*tenet.Data, error) {

	// 1. Request parent event
	rootEvent, err := s.service.RequestEvent(ctx, code)
	if err != nil {
		return nil, err
	}

	data := &tenet.Data{
		Event: tenet.EnhancedEvent{
			Event:  rootEvent,
			Relays: []string{},
		},
	}

	npub, _ := nip19.EncodePublicKey(rootEvent.PubKey)
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
		data.TemplateId = tenet.ListArticle
		events, err := s.service.AuthorArticles(ctx, npub)
		if err != nil {
			return nil, err
		}
		s.log.Info("articles pulled as children", "count", len(events))
		// TODO: Populate data.Notes with the list of requested articles.
		// This will be rendered using the ListArticle template.
		for _, e := range events {
			data.Notes = append(data.Notes, tenet.EnhancedEvent{Event: e})
		}
	case 30023:

		data.TemplateId = tenet.Article
		data.Content = mdToHtml(rootEvent.Content)

		// 1. Process a list of kind 9082
		// 2. Use the first identifier of the article to request highlight data
		// 3. Add the highlights to the data.Notes list
		if d := rootEvent.Tags.GetFirst([]string{"d", ""}); d != nil {

			highlights, err := s.service.ArticleHighlights(ctx, rootEvent.Kind, rootEvent.PubKey, d.Value())
			if err != nil {
				return nil, err
			}

			s.log.Info("articles pulled as children", "count", len(highlights))
		}

		// TODO add highlights to content string

	default:
		data.TemplateId = tenet.Unkown
	}

	return data, nil
}
