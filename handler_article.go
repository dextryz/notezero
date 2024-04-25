package notezero

import (
	"html/template"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
)

// 1. Highlights are encoded into data.Notes
// 2. Process the highlights into the data.Content string
func (s *Handler) ContentHandler(w http.ResponseWriter, r *http.Request) {

	code := r.PathValue("naddr")

	data, err := s.requestData(r.Context(), code, 0, true)
	if err != nil {
		s.log.Error("failed to get events", slog.Any("error", err))
		http.Error(w, "failed to get counts", http.StatusInternalServerError)
		return
	}

	var component templ.Component

	switch data.TemplateId {
	case Article:
		component = ContentTemplate(ArticleParams{
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
