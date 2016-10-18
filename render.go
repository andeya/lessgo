package lessgo

import (
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/lessgo/lessgo/pongo2"
)

type (
	Tpl struct {
		template *pongo2.Template
		modTime  time.Time
	}
	// Pongo2Render is a custom lessgo template renderer using Pongo2.
	Pongo2Render struct {
		set        *pongo2.TemplateSet
		caching    bool // false=disable caching, true=enable caching
		tplCache   map[string]*Tpl
		tplContext pongo2.Context // Context hold globle func for tpl
		sync.RWMutex
	}
)

// New creates a new Pongo2Render instance with custom Options.
func NewPongo2Render(caching bool) *Pongo2Render {
	return &Pongo2Render{
		set:        pongo2.NewSet("lessgo", pongo2.DefaultLoader),
		caching:    caching,
		tplCache:   make(map[string]*Tpl),
		tplContext: make(pongo2.Context),
	}
}

func (p *Pongo2Render) TemplateVariable(name string, v interface{}) {
	switch d := v.(type) {
	case func(in *pongo2.Value, param *pongo2.Value) (out *pongo2.Value, err *pongo2.Error):
		pongo2.RegisterFilter(name, d)
	case pongo2.FilterFunction:
		pongo2.RegisterFilter(name, d)
	default:
		p.tplContext[name] = d
	}
}

// Render should render the template to the io.Writer.
func (p *Pongo2Render) Render(w io.Writer, filename string, data interface{}, c *Context) error {
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

	for k, v := range p.tplContext {
		if _, ok := data2[k]; !ok {
			data2[k] = v
		}
	}

	if p.caching {
		template = pongo2.Must(p.FromCache(filename))
	} else {
		template = pongo2.Must(p.set.FromFile(filename))
	}
	return template.ExecuteWriter(data2, w)
}

func (p *Pongo2Render) FromCache(fname string) (*pongo2.Template, error) {
	//从文件系统缓存中获取文件信息
	fbytes, finfo, exist := lessgo.App.MemoryCache().GetCacheFile(fname)

	// 文件已不存在
	if !exist {
		// 移除模板中缓存
		p.Lock()
		_, has := p.tplCache[fname]
		if has {
			delete(p.tplCache, fname)
		}
		p.Unlock()
		// 返回错误
		return nil, errors.New(fname + "is not found.")
	}

	// 查看模板缓存
	p.RLock()
	tpl, has := p.tplCache[fname]
	p.RUnlock()

	// 存在模板缓存且文件未更新时，直接读模板缓存
	if has && p.tplCache[fname].modTime.Equal(finfo.ModTime()) {
		return tpl.template, nil
	}

	// 缓存模板不存在或文件已更新时，均新建缓存模板
	p.Lock()
	defer p.Unlock()

	// 创建新模板并缓存
	newtpl, err := p.set.FromBytes(fname, fbytes)
	if err != nil {
		return nil, err
	}

	p.tplCache[fname] = &Tpl{template: newtpl, modTime: finfo.ModTime()}
	return newtpl, nil
}
