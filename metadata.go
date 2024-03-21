package notezero

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

type ProfileMetadata struct {
	PubKey     string `json:"pubkey,omitempty"`
	Name       string `json:"name,omitempty"`
	About      string `json:"about,omitempty"`
	Website    string `json:"website,omitempty"`
	Picture    string `json:"picture,omitempty"`
	Banner     string `json:"banner,omitempty"`
	Identifier string `json:"identifier,omitempty"`
}

func (s ProfileMetadata) String() string {
	bytes, err := json.Marshal(s)
	if err != nil {
		log.Fatalln("Unable to convert event to string")
	}
	return string(bytes)
}

func (s ProfileMetadata) Npub() string {
	npub, _ := nip19.EncodePublicKey(s.PubKey)
	return npub
}

func ParseMetadata(e nostr.Event) (*ProfileMetadata, error) {

	if e.Kind != nostr.KindProfileMetadata {
		return nil, fmt.Errorf("event %s is kind %d, not 0", e.ID, e.Kind)
	}

	var profile ProfileMetadata
	err := json.Unmarshal([]byte(e.Content), &profile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata from event %s: %w", e.ID, err)
	}

	return &profile, nil
}
