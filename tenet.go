package tenet

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nbd-wtf/go-nostr"
)

type Highlight struct {
	Id        string `json:"id"`     // Event ID
	PubKey    string `json:"pubkey"` // Author who signed the highlight
	Content   string `json:"content"`
	Context   string `json:"context"`
	CreatedAt string `json:"created_at"`
	Event     string `json:"event"`
	Article   string `json:"article"` // 30032:pub:identifier
	Url       string `json:"url"`     // https://example.com
}

func ParseHighlight(e nostr.Event) (Highlight, error) {

	// Event information
	h := Highlight{
		Id:      e.ID,
		PubKey:  e.PubKey,
		Content: e.Content,
	}

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
		}
		if t.Key() == "r" {
			h.Url = t.Value()
		}
	}

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
