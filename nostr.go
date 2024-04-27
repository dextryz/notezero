package notezero

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/dextryz/notezero/badger"
	"github.com/fiatjaf/eventstore"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
	"gocloud.dev/blob"
	"gocloud.dev/blob/fileblob"
)

var eventMap = make(map[string]bool)

var defaultImage = "https://pfp.nostr.build/dfc3716d64302de9edff417fb561aae1ee90fc109acb8fc82839e580868cf34d.jpg"

type Nostr struct {
	db     eventstore.Store
	bucket *blob.Bucket
	imgDir string
	cache  *badger.Cache
	relays []string
}

func NewNostr(db eventstore.Store, dir string, cache *badger.Cache, relays []string) Nostr {

	b, err := fileblob.OpenBucket(dir, nil)
	if err != nil {
		log.Fatalln(err)
	}

	return Nostr{
		db:     db,
		bucket: b,
		imgDir: dir,
		cache:  cache,
		relays: relays,
	}
}

func (s *Nostr) Close() {
	err := s.bucket.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

func (s Nostr) pullProfileList(ctx context.Context, npubs []string) ([]*nostr.Event, error) {

	filter := nostr.Filter{
		Kinds: []int{nostr.KindProfileMetadata},
		Limit: 500,
	}

	for _, npub := range npubs {
		_, pk, err := nip19.Decode(npub)
		if err != nil {
			return nil, err
		}
		filter.Authors = append(filter.Authors, pk.(string))
	}

	pool := nostr.NewSimplePool(context.Background())

	latestNotes := func() <-chan *nostr.Event {
		notes := make(chan *nostr.Event)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			defer fmt.Println("latestNotes producer exited")
			defer close(notes)
			ch := pool.SubManyEose(ctx, s.relays, nostr.Filters{filter})
			for {
				select {
				case ie, more := <-ch:
					if !more {
						return
					}
					notes <- ie.Event
					s.db.SaveEvent(ctx, ie.Event)
				case <-ctx.Done():
					return
				}
			}
		}()
		return notes
	}

	profiles, err := eventstore.RelayWrapper{Store: s.db}.QuerySync(ctx, filter)
	if err != nil {
		return nil, err
	}

	if len(profiles) < len(CURATED_LIST) {
		for n := range latestNotes() {
			profiles = append(profiles, n)
		}
	}

	return profiles, nil
}

func (s Nostr) pullNextArticlePage(ctx context.Context, npubs []string, page int) ([]*nostr.Event, error) {

	filter := nostr.Filter{
		Kinds: []int{nostr.KindArticle},
		Limit: page * pageLimit,
	}

	for _, npub := range npubs {
		_, pk, err := nip19.Decode(npub)
		if err != nil {
			return nil, err
		}
		filter.Authors = append(filter.Authors, pk.(string))
	}

	pool := nostr.NewSimplePool(context.Background())

	latestNotes := func(done <-chan struct{}) <-chan *nostr.Event {
		notes := make(chan *nostr.Event)
		go func() {

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			defer fmt.Println("latestNotes producer exited")
			defer close(notes)

			ch := pool.SubManyEose(ctx, s.relays, nostr.Filters{filter})

			for {
				select {
				case ie, more := <-ch:
					if !more {
						return
					}

					_, ok := s.cache.Get(ie.GetID())
					if !ok {

						url, name := imageDetails(ie.Event, s.imgDir)

						err := s.SaveImage(url)
						if err != nil {
							log.Fatalln(err)
							return
						}

						err = s.db.SaveEvent(ctx, ie.Event)
						if err != nil {
							log.Fatalln(err)
							return
						}

						s.cache.Set(ie.GetID(), []byte(name))

						notes <- ie.Event
					}

				case <-ctx.Done():
					return
				case <-done:
					fmt.Println("reutrn case done")
					return
				}
			}
		}()
		return notes
	}

	// Fetch from local store if available
	lastNotes, err := eventstore.RelayWrapper{Store: s.db}.QuerySync(ctx, filter)
	if err != nil {
		return nil, err
	}

	notes := []*nostr.Event{}
	i := (page - 1) * pageLimit
	for i < len(lastNotes) {
		notes = append(notes, lastNotes[i])
		i++
	}

	count := len(notes)
	done := make(chan struct{})
	noteStream := latestNotes(done)
	for count < pageLimit {
		n := <-noteStream
		if n != nil {
			notes = append(notes, n)
			count++
		} else {
			break
		}
	}
	close(done)

	slog.Info("notes pulled from relay set or cache", "count", count)

	slices.SortFunc(notes, func(a, b *nostr.Event) int { return int(b.CreatedAt - a.CreatedAt) })

	if len(notes) > 0 {
		pageUntil = notes[len(notes)-1].CreatedAt - 1
	}

	return notes, nil
}

func (s Nostr) SaveImage(url string) error {

	ctx := context.Background()

	slog.Info("saving image to blob storage", "url", url)

	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	// TODO: THis breaks my server when image is no longer available at src.
	if res.StatusCode > 299 {
		return fmt.Errorf("response failed with status code: %d and body: %s", res.StatusCode, body)
	}

	name := path.Base(url)
	if len(name) > 64 {
		name = name[:64]
	}

	err = s.bucket.WriteAll(ctx, name, body, nil)
	if err != nil {
		return err
	}

	slog.Info("image stored to bucket", "name", name, "url", url)

	return nil
}

func imageDetails(e *nostr.Event, imgDir string) (url, name string) {
	for _, t := range e.Tags {
		if t.Key() == "image" {
			url = t.Value()
		}
	}
	if strings.Split(url, ":")[0] != "https" || url == "" {
		url = defaultImage
	}

	name = path.Base(url)
	if len(name) > 64 {
		name = name[:64]
	}

	name = fmt.Sprintf("%s/%s", imgDir, name)

	imageFiletype(name)

	return url, name
}

func imageFiletype(name string) {
	file, err := os.Open(name)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Read the first 512 bytes of the file to pass to DetectContentType.
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	// Use http.DetectContentType to determine the content type.
	contentType := http.DetectContentType(buffer)
	fmt.Println("Detected content type:", contentType)
}
