package markdown

import (
	"io"
	"text/template"
)

const tpl = `
<html>
<head>
<style>
{{.css}}
</style>
</head>

<body>
<article class="markdown-body">
{{.body}}
</article>
</body>

</html>
`

const (
	githubCommonHTMLFlags = 0 |
		HTML_USE_XHTML |
		HTML_USE_SMARTYPANTS |
		HTML_SMARTYPANTS_FRACTIONS |
		HTML_SMARTYPANTS_LATEX_DASHES

	githubCommonExtensions = 0 |
		EXTENSION_NO_INTRA_EMPHASIS |
		EXTENSION_TABLES |
		EXTENSION_FENCED_CODE |
		EXTENSION_AUTOLINK |
		EXTENSION_STRIKETHROUGH |
		EXTENSION_SPACE_HEADERS |
		EXTENSION_HEADER_IDS |
		EXTENSION_BACKSLASH_LINE_BREAK |
		EXTENSION_DEFINITION_LISTS
)

func GithubMarkdown(in []byte, out io.Writer, hasCatalog bool) error {
	flg := githubCommonHTMLFlags
	if hasCatalog {
		flg |= HTML_TOC
	}
	render := HtmlRenderer(flg, "", css)
	body := MarkdownOptions(in, render, Options{
		Extensions: githubCommonExtensions,
	})
	m := map[string]interface{}{
		"css":  css,
		"body": string(body),
	}
	return template.Must(template.New("markdown").Parse(tpl)).Execute(out, m)
}
