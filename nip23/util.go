package nip23

import (
	"log"
	"regexp"

	"github.com/dextryz/tenet"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// TODO: Maybe move this to core DL

func MdToHtml(a *tenet.Article) (tenet.Article, error) {

	text, err := ReplaceReferences(a.Content)
	if err != nil {
		log.Fatalln(err)
	}

	// create markdown parser with extensions
	extensions := parser.CommonExtensions
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse([]byte(text))

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	c := markdown.Render(doc, renderer)

	return tenet.Article{
		PubKey:     a.PubKey,
		Identifier: a.Identifier,
		Title:      a.Title,
		Content:    string(c),
		Tags:       a.Tags,
		Events:     a.Events,
		Urls:       a.Urls,
	}, nil
}

// text := "Click [me](nostr:nevent17915d512457e4bc461b54ba95351719c150946ed4aa00b1d83a263deca69dae) to"
// replacement := `<a href="#" hx-get="article/$2" hx-push-url="true" hx-target="body" hx-swap="outerHTML">$1</a>`
func ReplaceReferences(text string) (string, error) {

	// Define the regular expression pattern to match the markdown-like link
	//pattern := `\[(.*?)\]\((.*?)\)`
	pattern := `\[(.*?)\]\(nostr:(.*?)\)`

	// Compile the regular expression
	re := regexp.MustCompile(pattern)

	// Define the replacement pattern
	replacement := `<a href="#" class="inline"
        hx-get="$2"
        hx-push-url="true"
        hx-target="body"
        hx-swap="outerHTML">$1
    </a>`

	// Replace the matched patterns with the HTML tag
	result := re.ReplaceAllString(text, replacement)

	return result, nil
}
