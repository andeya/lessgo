package lessgo

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"
	// "github.com/lessgo/lessgo/utils"
)

func staticRoute() {
	// 注册固定的静态文件与目录
	DefLessgo.Echo.File("/favicon.ico", IMG_DIR+"/favicon.ico")
	DefLessgo.Echo.Static("/uploads", UPLOADS_DIR, autoHTMLSuffix())
	DefLessgo.Echo.Static("/static", STATIC_DIR, filterTemplate(), autoHTMLSuffix())
	DefLessgo.Echo.Static("/business", BUSINESS_VIEW_DIR, filterTemplate(), autoHTMLSuffix())
	DefLessgo.Echo.Static("/system", SYSTEM_VIEW_DIR, filterTemplate(), autoHTMLSuffix())

	// 注册模块中的静态目录
	// staticModule(BUSINESS_VIEW_DIR)
	// staticModule(SYSTEM_VIEW_DIR)
}

// func staticModule(root string) {
// 	dirs := utils.WalkDirs(root, VIEW_PKG)
// 	for _, dir := range dirs {
// 		DefLessgo.Echo.Static(urlPrefix(dir), dir, autoHTMLSuffix())
// 	}
// }

// func urlPrefix(dir string) string {
// 	urlprefix := strings.TrimSuffix(dir, VIEW_PKG)
// 	urlprefix = strings.Replace(urlprefix, MODULE_SUFFIX, "", -1)
// 	return "/" + strings.ToLower(urlprefix)
// }

func filterTemplate() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) (err error) {
			ext := path.Ext(c.Request().URL().Path())
			if len(ext) >= 4 && ext[:4] == TPL_EXT {
				return c.NoContent(http.StatusForbidden)
			}
			return next(c)
		}
	}
}

func autoHTMLSuffix() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) (err error) {
			p := c.Request().URL().Path()
			ext := path.Ext(p)
			if ext == "" || ext[0] != '.' {
				c.Request().URL().SetPath(strings.TrimSuffix(p, ext) + STATIC_HTML_EXT + ext)
				c.Object().pvalues[0] += STATIC_HTML_EXT
			}
			return next(c)
		}
	}
}

type (
	MemoryCache struct {
		enable  *bool
		filemap map[string]*Cachefile
		gc      time.Duration
		once    sync.Once
		sync.RWMutex
	}
	Cachefile struct {
		fname string
		info  os.FileInfo
		bytes []byte
		exist bool
	}
)

func NewMemoryCache(gc time.Duration) *MemoryCache {
	return &MemoryCache{
		enable:  new(bool),
		filemap: map[string]*Cachefile{},
		gc:      gc,
	}
}

func (c *Cachefile) Update() {
	file, err := os.Open(c.fname)
	if err != nil {
		return
	}
	defer file.Close()
	info, _ := file.Stat()
	var bufferWriter bytes.Buffer
	io.Copy(&bufferWriter, file)
	c.bytes = bufferWriter.Bytes()
	c.exist = true
	c.info = info
}

func (m *MemoryCache) Enable() bool {
	m.RLock()
	defer m.RUnlock()
	return *m.enable
}

func (m *MemoryCache) SetEnable(bl bool) {
	m.Lock()
	defer m.Unlock()
	if m.enable == nil {
		m.enable = &bl
		if bl {
			m.memoryCacheMonitor()
			return
		}
	}

	_bl := *m.enable
	if bl && _bl {
		return
	}
	*m.enable = bl
	if _bl {
		return
	}
	m.enable = &bl
	m.once = sync.Once{}
	m.memoryCacheMonitor()
}

func (m *MemoryCache) memoryCacheMonitor() {
	enable := m.enable
	go m.once.Do(func() {
		for *enable {
			time.Sleep(m.gc)
			m.RLock()
			for fname, cfile := range m.filemap {
				if !cfile.exist {
					cfile.Update()
					continue
				}
				info, err := os.Stat(fname)
				if err != nil {
					if os.IsNotExist(err) {
						cfile.exist = false
						cfile.bytes = nil
						cfile.info = nil
					}
					continue
				}
				if info.ModTime().Equal(cfile.info.ModTime()) {
					continue
				}
				cfile.Update()
			}
			m.RUnlock()
		}
	})
}

func (m *MemoryCache) GetCacheFile(fname string) (*bytes.Reader, os.FileInfo, bool) {
	cfile, ok := m.filemap[fname]
	if ok {
		return bytes.NewReader(cfile.bytes), cfile.info, cfile.exist
	}

	m.Lock()
	defer m.Unlock()

	cfile, ok = m.filemap[fname]
	if ok {
		return bytes.NewReader(cfile.bytes), cfile.info, cfile.exist
	}

	file, err := os.Open(fname)
	if err != nil {
		return nil, nil, false
	}
	defer file.Close()
	info, _ := file.Stat()
	var bufferWriter bytes.Buffer
	io.Copy(&bufferWriter, file)
	cfile = &Cachefile{
		fname: fname,
		bytes: bufferWriter.Bytes(),
		info:  info,
		exist: true,
	}
	m.filemap[fname] = cfile

	return bytes.NewReader(cfile.bytes), cfile.info, cfile.exist
}
