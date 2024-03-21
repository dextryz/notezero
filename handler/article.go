package handler

import (
	"html/template"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/dextryz/notezero"
	"github.com/dextryz/notezero/tmp"
)

// Poplated the data.Notes field with a list of requested notes based on the search field.
func (s *Handler) ArticleHandler(w http.ResponseWriter, r *http.Request) {

	code := r.PathValue("naddr")

	s.log.Info("handler for article", "naddr", code)

	data, err := s.requestData(r.Context(), code)
	if err != nil {
		s.log.Error("failed to get events", slog.Any("error", err))
		http.Error(w, "failed to get counts", http.StatusInternalServerError)
		return
	}

	s.log.Info("rendering article view", "author", data.Npub)

	var component templ.Component

	switch data.TemplateId {
	case notezero.Article:
		component = tmp.ArticleTemplate(notezero.ArticleParams{
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
