package lessgo

import (
	"encoding/json"
	"io"

	"github.com/lessgo/lessgo/pongo2"
)

type (
	// Pongo2Render is a custom lessgo template renderer using Pongo2.
	Pongo2Render struct {
		set   *pongo2.TemplateSet
		debug bool
	}
)

// New creates a new Pongo2Render instance with custom Options.
func NewPongo2Render(debug bool) *Pongo2Render {
	return &Pongo2Render{
		set:   pongo2.NewSet("lessgo", pongo2.DefaultLoader),
		debug: debug,
	}
}

// Render should render the template to the io.Writer.
func (p *Pongo2Render) Render(w io.Writer, filename string, data interface{}, c Context) error {
	var (
		template *pongo2.Template
		data2    = pongo2.Context{}
	)

	switch d := data.(type) {
	case pongo2.Context:
		data2 = d
	case map[string]interface{}:
		data2 = pongo2.Context(d)
	default:
		b, _ := json.Marshal(data)
		json.Unmarshal(b, &data2)
	}

	if p.debug {
		template = pongo2.Must(p.set.FromFile(filename))
	} else {
		template = pongo2.Must(p.set.FromCache(filename))
	}

	return template.ExecuteWriter(data2, w)
}
