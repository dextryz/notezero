package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/dextryz/tenet"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"

	"github.com/dextryz/tenet/component"
)

type HighlighterService interface {
	Request(ctx context.Context, naddr string) ([]*tenet.Highlight, error)
	RequestByNevent(ctx context.Context, nevent string) (*tenet.Highlight, error)
	ApplyToContent(ctx context.Context, a *tenet.Article) error
}

type ProfileService interface {
	Request(ctx context.Context, pubkey string) (tenet.Profile, error)
}

type ArticleService interface {
	Request(ctx context.Context, naddr string) (tenet.Article, error)
	RequestByNpub(ctx context.Context, npub string) ([]*tenet.Article, error)
}

type Handler struct {
	Log              *slog.Logger
	HighlightService HighlighterService
	ProfileService   ProfileService
	ArticleService   ArticleService
}

func New(
	log *slog.Logger,
	hs HighlighterService,
	ps ProfileService,
	as ArticleService,
) *Handler {

	return &Handler{
		Log:              log,
		HighlightService: hs,
		ProfileService:   ps,
		ArticleService:   as,
	}
}

func (s *Handler) Highlight(w http.ResponseWriter, r *http.Request) {

	nevent := r.PathValue("nevent")

	if nevent == "" {
		s.Log.Error("nevent is empty")
		http.Error(w, "failed to get counts", http.StatusInternalServerError)
		return
	}

	s.Log.Info("pulling hightlight", "nevent", nevent)

	h, err := s.HighlightService.RequestByNevent(r.Context(), nevent)
	if err != nil {
		s.Log.Error("failed to REQ highlights", slog.Any("error", err))
		http.Error(w, "failed to get counts", http.StatusInternalServerError)
		return
	}

	component.Highlight(*h).Render(r.Context(), w)
}

func (s *Handler) Nip19(w http.ResponseWriter, r *http.Request) {

	nuri := r.PathValue("nuri")

	prefix, data, err := nip19.Decode(nuri)
	if err != nil {
		panic(err)
	}

	if prefix == "naddr" {
		ep := data.(nostr.EntityPointer)
		switch ep.Kind {
		case nostr.KindArticle:
			s.articleHandler(w, r, nuri)
		case tenet.KindHighlight:
			s.Log.Error("not implemented")
		}
	} else {
		panic(fmt.Errorf("not a nostr URI: %s", nuri))
	}
}

func (s *Handler) ListArticles(w http.ResponseWriter, r *http.Request, npub string) {

	articles, err := s.ArticleService.RequestByNpub(r.Context(), npub)
	if err != nil {
		s.Log.Error("failed to REQ articles", slog.Any("error", err))
		http.Error(w, "failed to get counts", http.StatusInternalServerError)
		return
	}

	s.Log.Info("articles pulled", "count", len(articles))

	component.ArticleCard(articles).Render(r.Context(), w)
}

func (s *Handler) List(w http.ResponseWriter, r *http.Request) {

	search := r.URL.Query().Get("search")

	prefix, _, err := nip19.Decode(search)
	if err != nil {
		panic(err)
	}

	if prefix == "naddr" {
		s.ListHighlights(w, r, search)
	} else if prefix == "npub" {
		s.ListArticles(w, r, search)
	} else {
		panic(fmt.Errorf("not a nostr URI: %s", search))
	}
}

func (s *Handler) ListHighlights(w http.ResponseWriter, r *http.Request, naddr string) {

	// TODO: cache
	a, err := s.ArticleService.Request(r.Context(), naddr)
	if err != nil {
		s.Log.Error("failed to REQ articles", slog.Any("error", err))
		http.Error(w, "failed to get counts", http.StatusInternalServerError)
		return
	}

	s.Log.Info("pulling article hightlights", "naddr", naddr)

	highlights, err := s.HighlightService.Request(r.Context(), naddr)
	if err != nil {
		s.Log.Error("failed to REQ highlights", slog.Any("error", err))
		http.Error(w, "failed to get counts", http.StatusInternalServerError)
		return
	}

	s.Log.Info("highlights pulled", "count", len(highlights))

	// TODO: Use TEMPL to view
	for _, v := range highlights {

		p, err := s.ProfileService.Request(r.Context(), v.PubKey)
		if err != nil {
			s.Log.Error("failed to REQ profile", slog.Any("error", err), "naddr", naddr)
			http.Error(w, "failed to get counts", http.StatusInternalServerError)
			return
		}

		articleUrl := "articles/" + v.Naddr
		highlightUrl := "high/" + v.Id

		component.HighlightCard(*v, p, a, articleUrl, highlightUrl).Render(r.Context(), w)
	}
}

func (s *Handler) Article(w http.ResponseWriter, r *http.Request) {
	naddr := r.PathValue("naddr")
	s.articleHandler(w, r, naddr)
}

func (s *Handler) articleHandler(w http.ResponseWriter, r *http.Request, naddr string) {

	s.Log.Info("retrieving article from cache", "naddr", naddr)

	// TODO: Alrady REQ, should be in cache
	a, err := s.ArticleService.Request(r.Context(), naddr)
	if err != nil {
		s.Log.Error("failed to get events", slog.Any("error", err), "naddr", naddr)
		http.Error(w, "failed to get counts", http.StatusInternalServerError)
		return
	}

	err = s.HighlightService.ApplyToContent(r.Context(), &a)
	if err != nil {
		s.Log.Error("failed to get events", slog.Any("error", err), "naddr", naddr)
		http.Error(w, "failed to get counts", http.StatusInternalServerError)
		return
	}

	s.Log.Info("highlights applied to article content", "naddr", naddr)

	component.Article(a, a.Content).Render(r.Context(), w)
}

func (s *Handler) View(w http.ResponseWriter, r *http.Request) {
	component.Index().Render(r.Context(), w)
}
