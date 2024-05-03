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

						filepath, _, err := s.fetchImage(*ie.Event, s.imgDir)
						if err != nil {
							log.Fatalf("cannot store img to bucket: %v", err)
							return
						}

						err = s.db.SaveEvent(ctx, ie.Event)
						if err != nil {
							log.Fatalf("cannot save event to store: %v", err)
							return
						}

						s.cache.Set(ie.GetID(), []byte(filepath))

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

	stack := []*nostr.Event{}
	i := (page - 1) * pageLimit
	for i < len(lastNotes) {
		stack = append(stack, lastNotes[i])
		i++
	}

	count := len(stack)
	done := make(chan struct{})
	noteStream := latestNotes(done)
	for count < pageLimit {
		n := <-noteStream
		if n != nil {
			stack = append(stack, n)
			count++
		} else {
			break
		}
	}
	close(done)

	slog.Info("notes pulled from relay set or cache", "count", count)

	slices.SortFunc(stack, func(a, b *nostr.Event) int {
		return int(b.CreatedAt - a.CreatedAt)
	})

	// Check created time for last item (oldest) in stack.
	if len(stack) > 0 {
		pageUntil = stack[len(stack)-1].CreatedAt - 1
	}

	return stack, nil
}

func (s Nostr) fetchImage(e nostr.Event, imgDir string) (string, int64, error) {

	slog.Info("fetching img", "fn", "fetchImage", "eventId", e.GetID())

	// Extract image URL from event

	var url string
	for _, t := range e.Tags {
		if t.Key() == "image" {
			url = t.Value()
		}
	}
	if strings.Split(url, ":")[0] != "https" || url == "" {
		url = defaultImage
	}

	// Do the actual fetch

	res, err := http.Get(url)
	if err != nil {
		return "", 0, err
	}
	defer res.Body.Close()

	// Generate a new unique image name and store locally

	// TODO: rename using a UUID
	name := path.Base(url)
	if len(name) > 64 {
		name = name[:64]
	}
	filepath := fmt.Sprintf("%s/%s", imgDir, name)

	f, err := os.Create(filepath)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	n, err := io.Copy(f, res.Body)
	if err != nil {
		slog.Error("cannot gob write to file", "err", err)
	}

	slog.Info("image stored to bucket", "filepath", filepath, "byteCount", n)

	return filepath, n, nil
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
