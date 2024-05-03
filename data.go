package notezero

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nbd-wtf/go-nostr/nip19"
)

type TemplateID int

const (
	Profile TemplateID = iota
	ListArticle
	ListHighlight
	ListTodo
	Article
	Highlight
	Unkown
)

// Has a parent event and a set of children:
// 1. Parent is 30023 - Children is 9802
// 2. Parent is 0 - Children is 30023
type RawData struct {
	TemplateId TemplateID
	Event      EnhancedEvent
	Metadata   ProfileMetadata // Obviously always needs the author profile data
	Notes      []EnhancedEvent // For example, if the parent is an article, the children will be highlights
	PubKey     string
	Npub       string
	Naddr      string
	NaddrNaked string
	CreatedAt  string
	ModifiedAt string
	Kind       string
	Content    string
}

var profiles = make(map[string]*ProfileMetadata)

// FIXME: Remove the content bool hack
func (s *Handler) processPrompt(ctx context.Context, code string, page int, content bool) (*RawData, error) {

	codes := strings.Split(code, ":")

	var prefix string
	if len(codes) > 1 {
		prefix = codes[0]
		code = codes[1]
	}

	s.log.Info("process prompt code", "prefix", prefix, "code", code)

	// 1. Request parent event
	rootEvent, err := s.service.RequestEvent(ctx, code)
	if err != nil {
		return nil, err
	}

	// 2. Get the owner of this root event.
	npub, _ := nip19.EncodePublicKey(rootEvent.PubKey)
	profileEvent, err := s.service.Profile(ctx, npub)
	if err != nil {
		return nil, err
	}
	metadata, err := ParseMetadata(*profileEvent)
	if err != nil {
		return nil, err
	}

	data := &RawData{
		Metadata: *metadata,
		Event: EnhancedEvent{
			Event:  rootEvent,
			Relays: []string{},
		},
	}
	data.TemplateId = Unkown
	data.CreatedAt = rootEvent.CreatedAt.Time().String()
	data.PubKey = rootEvent.PubKey
	data.Npub = npub // later
	data.Naddr = ""
	data.NaddrNaked = ""
	data.CreatedAt = time.Unix(int64(rootEvent.CreatedAt), 0).Format("2006-01-02 15:04:05")
	data.ModifiedAt = time.Unix(int64(rootEvent.CreatedAt), 0).Format("2006-01-02T15:04:05Z07:00")

	if rootEvent.Kind >= 30000 && rootEvent.Kind < 40000 {
		if d := rootEvent.Tags.GetFirst([]string{"d", ""}); d != nil {
			data.NaddrNaked, err = nip19.EncodeEntity(rootEvent.PubKey, rootEvent.Kind, d.Value(), nil)
			if err != nil {
				return nil, err
			}
		}
	}

	// 3. Populate the children
	switch rootEvent.Kind {
	case 0:

		// If prompt is not profile:npub.., but only npub, then pull articles too
		switch prefix {
		//case "article":
		case "":

			events, err := s.service.AuthorArticles(ctx, npub)
			if err != nil {
				return nil, err
			}

			for _, e := range events {
				note := EnhancedEvent{
					Event:   e,
					Profile: metadata,
				}
				data.Notes = append(data.Notes, note)
			}

			data.TemplateId = ListArticle
		case "profile":
			// TODO replave with nprofile code
			data.TemplateId = Profile
		case "highlight":
			fmt.Println("list author highlights")
		case "todo":
			fmt.Println("list author todos")
		}

	case 30023:

		data.TemplateId = Article
		data.Content = mdToHtml(rootEvent.Content)

		if content {

			// TODO: Update to use channel pipeline pattern

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
				}

				intervals := highlightIntervals(data.Content, highlights)
				merged := mergeIntervals(intervals)
				data.Content = highlight(data.Content, merged)
			}
		}
	default:
		//		data.TemplateId = Unkown
	}

	return data, nil
}
