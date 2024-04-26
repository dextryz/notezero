package notezero

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

type EnhancedEvent struct {
	*nostr.Event
	Profile *ProfileMetadata
	Relays  []string
}

func (s EnhancedEvent) ImageUrl() string {
	var res string
	for _, t := range s.Tags {
		if t.Key() == "image" {
			res = t.Value()
		}
	}
	if strings.Split(res, ":")[0] != "https" {
		return ""
	}
	return res
}

func (s EnhancedEvent) ImageName() string {
	return path.Base(s.ImageUrl())
}

func (s EnhancedEvent) Image() string {
	var res string
	for _, t := range s.Tags {
		if t.Key() == "image" {
			res = t.Value()
		}
	}
	return res
}

func (s EnhancedEvent) Title() string {
	var res string
	for _, t := range s.Tags {
		if t.Key() == "title" {
			res = t.Value()
		}
	}
	return res
}

func (s EnhancedEvent) HashTags() []string {
	tags := []string{}
	for _, t := range s.Tags {
		// 		if t.Key() == "title" {
		// 			a.Title = t.Value()
		// 		}
		// 		if t.Key() == "d" {
		// 			a.Identifier = t.Value()
		// 		}
		// 		if t.Key() == "e" {
		// 			a.Events = append(a.Events, t.Value())
		// 		}
		// 		if t.Key() == "r" {
		// 			a.Urls = append(a.Urls, t.Value())
		// 		}

		if t.Key() == "t" {
			tags = append(tags, t.Value())
		}
	}
	return tags
}

func (s EnhancedEvent) Naddr() string {

	var identifier string
	for _, t := range s.Tags {
		if t.Key() == "d" {
			identifier = t.Value()
		}
	}

	naddr, _ := nip19.EncodeEntity(
		s.PubKey,
		nostr.KindArticle,
		identifier,
		[]string{}, // TODO: This worries me
	)
	return naddr
}

func (s EnhancedEvent) Npub() string {
	npub, _ := nip19.EncodePublicKey(s.PubKey)
	return npub
}

func (s EnhancedEvent) NpubShort() string {
	npub := s.Npub()
	return npub[:8] + "…" + npub[len(npub)-4:]
}

func (s EnhancedEvent) Nevent() string {
	nevent, _ := nip19.EncodeEvent(s.ID, s.Relays, s.PubKey)
	return nevent
}

func (s EnhancedEvent) CreatedAtStr() string {
	return time.Unix(int64(s.Event.CreatedAt), 0).Format("2006-01-02 15:04:05")
}

func (s EnhancedEvent) ModifiedAtStr() string {
	return time.Unix(int64(s.Event.CreatedAt), 0).Format("2006-01-02T15:04:05Z07:00")
}
