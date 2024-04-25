package notezero

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
	Page  int
}

type SpinnerParams struct {
	Id string
}

type ArticleParams struct {
	Event    EnhancedEvent
	Metadata ProfileMetadata
	Details  DetailsParams
	Content  template.HTML // Highlights are encoded into the content
}
