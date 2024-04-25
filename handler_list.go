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
}

func NewHandler(log *slog.Logger, es EventService) *Handler {
	return &Handler{
		log:     log,
		service: es,
	}
}

// Poplated the data.Notes field with a list of requested notes based on the search field.
func (s *Handler) CodeHandler(w http.ResponseWriter, r *http.Request) {

	code := r.PathValue("code")
	pageStr := r.URL.Query().Get("page")

	var page int
	if pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			panic(err)
		}
		page += 1
	}

	data, err := s.processPrompt(r.Context(), code, page, false)
	if err != nil {
		s.log.Error("failed to get events", slog.Any("error", err))
		http.Error(w, "failed to get counts", http.StatusInternalServerError)
		return
	}

	var component templ.Component

	switch data.TemplateId {
	case ListArticle:
		component = IndexTemplate(ListArticleParams{
			Notes: data.Notes,
			Page:  page,
		})
	case Article:
		component = ArticleTemplate(ArticleParams{
			Event:   data.Event,
			Content: template.HTML(data.Content), // data.Content is converted from Md to Html in data service.
		})
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
