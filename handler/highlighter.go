package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/dextryz/tenet"

	"github.com/dextryz/tenet/component"
)

type HighlighterService interface {
	Request(ctx context.Context, naddr string) ([]*tenet.Highlight, error)
}

type ProfileService interface {
	Request(ctx context.Context, pubkey string) (tenet.Profile, error)
}

type Handler struct {
	Log              *slog.Logger
	HighlightService HighlighterService
	ProfileService   ProfileService
}

func New(log *slog.Logger, hs HighlighterService, ps ProfileService) *Handler {
	return &Handler{
		Log:              log,
		HighlightService: hs,
		ProfileService:   ps,
	}
}

func (s *Handler) Get(w http.ResponseWriter, r *http.Request) {

	naddr := r.URL.Query().Get("naddr")

	s.Log.Info("pulling article hightlights", "naddr", naddr)

	highlights, err := s.HighlightService.Request(r.Context(), naddr)
	if err != nil {
		s.Log.Error("failed to get events", slog.Any("error", err))
		http.Error(w, "failed to get counts", http.StatusInternalServerError)
		return
	}

	s.Log.Info("highlights pulled", "count", len(highlights))

	// TODO: Use TEMPL to view
	for _, v := range highlights {

		p, err := s.ProfileService.Request(r.Context(), v.PubKey)
		if err != nil {
			s.Log.Error("failed to get events", slog.Any("error", err))
			http.Error(w, "failed to get counts", http.StatusInternalServerError)
			return
		}

		component.Card(*v, p).Render(r.Context(), w)
	}
}

func (s *Handler) View(w http.ResponseWriter, r *http.Request) {
	component.Index().Render(r.Context(), w)
}
