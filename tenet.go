package tenet

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

const (
	KindHighlight       int = 9802
	KindImageGeneration int = 5100
)

type Config struct {
	Relays []string `json:"relays"`
}

func LoadConfig(path string) (*Config, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Config file: %v", err)
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		log.Fatal(err)
	}

	return &cfg, nil
}

type Article struct {
	Naddr          string   `json:"naddr"`  // Event ID
	PubKey         string   `json:"pubkey"` // Author who signed the highlight
	Identifier     string   `json:"identifier"`
	Title          string   `json:"title"`
	Content        string   `json:"content"`
	PublishedAt    string   `json:"published_at"`
	Tags           []string `json:"tags"`
	Urls           []string `json:"urls"`
	Events         []string `json:"events"`
	HighlightCount string   `json:"highlight_count"`
	HighlightAuthors string   `json:"highlight_authors"`
}

func ParseArticle(e nostr.Event) (Article, error) {

	a := Article{
		PubKey:         e.PubKey,
		Content:        e.Content,
		PublishedAt:    e.CreatedAt.Time().String(),
		HighlightCount: "0",
		HighlightAuthors: "0",
	}

	for _, t := range e.Tags {
		if t.Key() == "title" {
			a.Title = t.Value()
		}
		if t.Key() == "d" {
			a.Identifier = t.Value()
		}
		// TODO: Check the # prefix and filter in tags.
		if t.Key() == "t" {
			a.Tags = append(a.Tags, t.Value())
		}
		if t.Key() == "e" {
			a.Events = append(a.Events, t.Value())
		}
		if t.Key() == "r" {
			a.Urls = append(a.Urls, t.Value())
		}
	}

	naddr, err := nip19.EncodeEntity(
		a.PubKey,
		nostr.KindArticle,
		a.Identifier,
		[]string{}, // TODO: This worries me
	)
	if err != nil {
		return a, nil
	}

	a.Naddr = naddr

	return a, nil
}

type Highlight struct {
	*Profile
	Id         string `json:"id"`     // Event ID
	Naddr      string `json:"naddr"`  // Event ID
	PubKey     string `json:"pubkey"` // Author who signed the highlight
	Content    string `json:"content"`
	Context    string `json:"context"`
	CreatedAt  string `json:"created_at"`
	Url        string `json:"url"` // https://example.com
	Event      string `json:"event"`
	Article    string `json:"article"`    // 30032:pub:identifier
	Identifier string `json:"identifier"` // dentifier
	Title      string `json:"title"`      // dentifier
}

func ParseHighlight(e nostr.Event) (Highlight, error) {

	// Event information
	h := Highlight{
		Id:      e.ID,
		PubKey:  e.PubKey,
		Content: e.Content,
	}

	// The pubkey of the article author (not the highlight author)
	var pubkey string

	// Add original source reference
	for _, t := range e.Tags {
		if t.Key() == "context" {
			h.Context = t.Value()
		}
		if t.Key() == "e" {
			h.Event = t.Value()
		}
		if t.Key() == "a" {
			h.Article = t.Value()
			h.Identifier = strings.Split(t.Value(), ":")[2]
			pubkey = strings.Split(t.Value(), ":")[1]
		}
		if t.Key() == "r" {
			h.Url = t.Value()
		}
	}

	naddr, err := nip19.EncodeEntity(
		pubkey,
		nostr.KindArticle,
		h.Identifier,
		[]string{}, // TODO: This worries me
	)
	if err != nil {
		return h, err
	}

	h.Naddr = naddr

	return h, nil
}

// TODO Check marshaling
type Profile struct {
	PubKey     string `json:"pubkey,omitempty"`
	Name       string `json:"name,omitempty"`
	About      string `json:"about,omitempty"`
	Website    string `json:"website,omitempty"`
	Picture    string `json:"picture,omitempty"`
	Banner     string `json:"banner,omitempty"`
	Identifier string `json:"identifier,omitempty"`
}

func (s Profile) String() string {
	bytes, err := json.Marshal(s)
	if err != nil {
		log.Fatalln("Unable to convert event to string")
	}
	return string(bytes)
}

func ParseMetadata(e nostr.Event) (*Profile, error) {

	if e.Kind != nostr.KindProfileMetadata {
		return nil, fmt.Errorf("event %s is kind %d, not 0", e.ID, e.Kind)
	}

	var profile Profile
	err := json.Unmarshal([]byte(e.Content), &profile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata from event %s: %w", e.ID, err)
	}

	return &profile, nil
}
