package tenet

import (
	"html/template"

	"github.com/a-h/templ"
)

type DetailsParams struct {
	HideDetails     bool
	CreatedAt       string
	EventJSON       template.HTML
	Metadata        ProfileMetadata
	Nevent          string
	Nprofile        string
	SeenOn          []string
	Kind            int
	KindNIP         string
	KindDescription string
	Extra           templ.Component
}

type BaseEventPageParams struct {
}

type ListArticleParams struct {
	Notes []EnhancedEvent
}

type ArticleParams struct {
	Event    EnhancedEvent
	Metadata ProfileMetadata
	Details  DetailsParams
	Content  template.HTML
}
