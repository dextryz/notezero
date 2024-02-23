package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/dextryz/nip84"

	"github.com/dextryz/tenet/component"
)

type HighlighterService interface {
	Request(ctx context.Context, naddr string) ([]*nip84.Highlight, error)
}

type Handler struct {
	Log     *slog.Logger
	Service HighlighterService
}

func New(log *slog.Logger, srv HighlighterService) *Handler {
	return &Handler{
		Log:     log,
		Service: srv,
	}
}

func (s *Handler) Get(w http.ResponseWriter, r *http.Request) {

	naddr := r.URL.Query().Get("naddr")

	s.Log.Info("pulling article hightlights for %s", naddr)

	highlights, err := s.Service.Request(r.Context(), naddr)
	if err != nil {
		s.Log.Error("failed to get events", slog.Any("error", err))
		http.Error(w, "failed to get counts", http.StatusInternalServerError)
		return
	}

	s.Log.Info("%d highlights pulled", len(highlights))

	// TODO: Use TEMPL to view
	component.Card(*highlights[0]).Render(r.Context(), w)
}

func (s *Handler) View(w http.ResponseWriter, r *http.Request) {
	component.Index().Render(r.Context(), w)
}
