package notezero

import (
	"html/template"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
)

type Handler struct {
	log     *slog.Logger
	service EventService
	ns      Nostr
}

func NewHandler(log *slog.Logger, es EventService, ns Nostr) *Handler {
	return &Handler{
		log:     log,
		service: es,
		ns:      ns,
	}
}

// Poplated the data.Notes field with a list of requested notes based on the search field.
func (s *Handler) CodeHandler(w http.ResponseWriter, r *http.Request) {

	page := 1
	pageStr := r.URL.Query().Get("page")
	if pageStr != "" {
		page, _ = strconv.Atoi(pageStr)
	}

	code := r.PathValue("code")
	if code == "" {
		s.HomepageHandler(w, r)
		return
	}

	// 1. Process the prompt code to get the raw data from a root event

	data, err := s.processPrompt(r.Context(), code, page, false)
	if err != nil {
		s.log.Error("failed to get events", slog.Any("error", err))
		http.Error(w, "failed to get counts", http.StatusInternalServerError)
		return
	}

	// 2. Process this raw data into structured front-end components to be viewed

	var component templ.Component
	switch data.TemplateId {
	case ListArticle:
		component = IndexTemplate(ListArticleParams{
			Notes: data.Notes,
			Code:  code,
			Page:  page,
		})
	case Article:
		component = ArticleTemplate(ArticleParams{
			Event:   data.Event,
			Content: template.HTML(data.Content), // data.Content is converted from Md to Html in data service.
		})
	default:
		s.log.Error("unable to render template", "func", "CodeHandler", "templateId", data.TemplateId, "code", code)
		http.Error(w, "tried to render an unsupported template", 500)
		return
	}

	// 3. Render the view components on the client side.

	err = component.Render(r.Context(), w)
	if err != nil {
		s.log.Error("error rendering tmpl", "error", err.Error())
	}
}
