package notezero

import (
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/microcosm-cc/bluemonday"
)

// TODO: Maybe move this to core DL
// TODO make links/reference optional: mdToHtml(conent string, skipLinks bool)

var mdrenderer = html.NewRenderer(html.RendererOptions{
	Flags: html.CommonFlags | html.HrefTargetBlank,
})

func stripLinksFromMarkdown(md string) string {
	// Regular expression to match Markdown links and HTML links
	linkRegex := regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)|<a[^>]*>(.*?)</a>`)

	// Replace both Markdown and HTML links with just the link text
	strippedMD := linkRegex.ReplaceAllString(md, "$1$2")

	return strippedMD
}

var tgivmdrenderer = html.NewRenderer(html.RendererOptions{
	Flags: html.CommonFlags | html.HrefTargetBlank,
	RenderNodeHook: func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
		// telegram instant view really doesn't like when there is an image inside a paragraph (like <p><img></p>)
		// so we use this custom thing to stop all paragraphs before the images, print the images then start a new
		// paragraph afterwards.
		if img, ok := node.(*ast.Image); ok {
			if entering {
				src := img.Destination
				w.Write([]byte(`</p><img src="`))
				html.EscLink(w, src)
				w.Write([]byte(`" alt="`))
			} else {
				if img.Title != nil {
					w.Write([]byte(`" title="`))
					html.EscapeHTML(w, img.Title)
				}
				w.Write([]byte(`" /><p>`))
			}
			return ast.GoToNext, true
		}
		return ast.GoToNext, false
	},
})

func sanitizeXSS(html string) string {
	p := bluemonday.UGCPolicy()
	p.AllowStyling()
	p.RequireNoFollowOnLinks(false)
	p.AllowElements("video", "source", "iframe")
	p.AllowAttrs("controls", "width").OnElements("video")
	p.AllowAttrs("src", "width").OnElements("source")
	p.AllowAttrs("src", "frameborder").OnElements("iframe")
	return p.Sanitize(html)
}
func markdownToHtml(md string, usingTelegramInstantView bool, skipLinks bool) string {
	md = strings.ReplaceAll(md, "\u00A0", " ")

	// create markdown parser with extensions
	// this parser is stateful so it must be reinitialized every time
	doc := parser.NewWithExtensions(
		parser.CommonExtensions |
			parser.AutoHeadingIDs |
			parser.NoEmptyLineBeforeBlock |
			parser.Footnotes,
	).Parse([]byte(md))

	renderer := mdrenderer
	if usingTelegramInstantView {
		renderer = tgivmdrenderer
	}

	// create HTML renderer with extensions
	output := string(markdown.Render(doc, renderer))

	if skipLinks {
		output = stripLinksFromMarkdown(output)
	}

	// sanitize content
	output = sanitizeXSS(output)

	return output
}

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

func applyHighlight(content, highlight string) string {

	fmt.Println("------------ Content")
	fmt.Println(content)
	fmt.Println("------------ Highliths")
	fmt.Println(highlight)
	fmt.Println("------------")

	if strings.Contains(content, highlight) {

		//         replace := `<span class="inline"
		//             hx-get="highlight/%s"
		//             hx-push-url="true"
		//             hx-target="body"
		//             hx-swap="outerHTML">%s
		//         </span>`
		//
		// 		txt := fmt.Sprintf(replace, h.Id, h.Content)

		txt := fmt.Sprintf("<span class='highlight'>%s</span>", highlight)
		content = strings.ReplaceAll(content, highlight, txt)
	}

	txt := fmt.Sprintf("<span class='highlight'>%s</span>", highlight)
	content = strings.ReplaceAll(content, highlight, txt)
	return content
}
