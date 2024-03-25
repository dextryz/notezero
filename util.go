package notezero

import (
	"log"
	"regexp"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// TODO: Maybe move this to core DL
// TODO make links/reference optional: mdToHtml(conent string, skipLinks bool)

func mdToHtml(content string) string {

	text, err := ReplaceReferences(content)
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

	res := markdown.Render(doc, renderer)

	return string(res)
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
