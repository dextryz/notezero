package notezero

type TemplateID int

const (
	Profile TemplateID = iota
	Article
	ListArticle
	Highlight
	ListHighlight
	Unkown
)

// Has a parent event and a set of children:
// 1. Parent is 30023 - Children is 9802
// 2. Parent is 0 - Children is 30023
type Data struct {
	TemplateId TemplateID
	Event      EnhancedEvent
	Metadata   ProfileMetadata // Obviously always needs the author profile data
	Notes      []EnhancedEvent // For example, if the parent is an article, the children will be highlights
	Npub       string
	Naddr      string
	NaddrNaked string
	CreatedAt  string
	ModifiedAt string
	Kind       string
	Content    string
}
