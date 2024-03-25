package notezero

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
)

type Handler struct {
	log     *slog.Logger
	service EventService
}

func NewHandler(log *slog.Logger, es EventService) *Handler {
	return &Handler{
		log:     log,
		service: es,
	}
}

func (s *Handler) Homepage(w http.ResponseWriter, r *http.Request) {
	IndexTemplate().Render(r.Context(), w)
}

// Poplated the data.Notes field with a list of requested notes based on the search field.
func (s *Handler) CodeHandler(w http.ResponseWriter, r *http.Request) {

	code := r.PathValue("code")

	fmt.Printf("Handler: %s\n", code)

	data, err := s.requestData(r.Context(), code, false)
	if err != nil {
		s.log.Error("failed to get events", slog.Any("error", err))
		http.Error(w, "failed to get counts", http.StatusInternalServerError)
		return
	}

	s.log.Info("requested data from nostr relays", "author", data.Npub, "noteCount", len(data.Notes))

	var component templ.Component

	// 1. A list of articles are returned is the search field was npub
	// 2. A list of highlights are returned is the search field was nevent of kind 30023
	switch data.TemplateId {
	case ListArticle:
		component = ListArticleTemplate(ListArticleParams{
			Notes: data.Notes,
		})
		fmt.Println("Component")
		fmt.Println(len(data.Notes))
	default:
		s.log.Error("unable to render template", "templateId", data.TemplateId)
		http.Error(w, "tried to render an unsupported template", 500)
		return
	}

	err = component.Render(r.Context(), w)
	if err != nil {
		s.log.Error("error rendering tmpl", "error", err.Error())
	}
}
