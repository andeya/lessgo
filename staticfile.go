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

// 注册固定的静态文件与目录
func staticRoute() {
	StaticBaseRouter("/uploads", UPLOADS_DIR, autoHTMLSuffix())
	StaticBaseRouter("/static", STATIC_DIR, filterTemplate(), autoHTMLSuffix())
	StaticBaseRouter("/static/img", IMG_DIR)
	StaticBaseRouter("/static/js", JS_DIR)
	StaticBaseRouter("/static/css", CSS_DIR)
	StaticBaseRouter("/static/plugin", PLUGIN_DIR)
	StaticBaseRouter("/bus", BUSINESS_VIEW_DIR, filterTemplate(), autoHTMLSuffix())
	StaticBaseRouter("/sys", SYSTEM_VIEW_DIR, filterTemplate(), autoHTMLSuffix())

	FileBaseRouter("/favicon.ico", IMG_DIR+"/favicon.ico")
}

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
		usedSize        int64 // 已用容量
		singleFileAllow int64 // 允许的最大文件
		maxCap          int64 // 最大缓存总量
		enable          *bool
		gc              time.Duration // 缓存更新检查时长及动态过期时长
		filemap         map[string]*Cachefile
		once            sync.Once
		sync.RWMutex
	}
	Cachefile struct {
		fname string
		info  os.FileInfo
		bytes []byte
		last  time.Time
		exist bool
	}
)

const (
	_cancache     = iota // 文件可被缓存
	_canupdate           // 文件缓存可被更新
	_unknown             // 文件状态为知
	_notexist            // 文件不存在
	_notallowed          // 文件大小超出允许范围
	_willoverflow        // 将超出缓存空间
)

func NewMemoryCache(singleFileAllow, maxCap int64, gc time.Duration) *MemoryCache {
	return &MemoryCache{
		singleFileAllow: singleFileAllow,
		maxCap:          maxCap,
		gc:              gc,
		enable:          new(bool),
		filemap:         map[string]*Cachefile{},
	}
}

func (c *Cachefile) Size() int64 {
	return int64(len(c.bytes))
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

func (m *MemoryCache) GetCacheFile(fname string) (*bytes.Reader, os.FileInfo, bool) {
	m.RLock()
	cfile, ok := m.filemap[fname]
	if ok {
		m.RUnlock()
		return bytes.NewReader(cfile.bytes), cfile.info, cfile.exist
	}
	m.RUnlock()

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
	// 检查是否加入缓存
	if size := m.usedSize + info.Size(); size <= m.maxCap {
		m.filemap[fname] = &Cachefile{
			fname: fname,
			bytes: bufferWriter.Bytes(),
			info:  info,
			exist: true,
			last:  time.Now(),
		}
		m.usedSize = size
	}
	return bytes.NewReader(bufferWriter.Bytes()), info, true
}

func (m *MemoryCache) memoryCacheMonitor() {
	enable := m.enable
	go m.once.Do(func() {
		for *enable {
			time.Sleep(m.gc)
			m.RLock()
			expired, mustExpired := []string{}, []string{}
			for fname, cfile := range m.filemap {
				// 检查缓存超时，超时则加入过期列表
				if cfile.last.Add(m.gc).Before(time.Now()) {
					expired = append(expired, fname)
					continue
				}

				// 获取本地文件状态
				info, status := m.check(fname)

				// 缓存内容被标记不存在时
				if !cfile.exist {
					// 检查本地文件是否可被缓存
					switch status {
					case _notexist:
					case _notallowed, _willoverflow:
						// 本地文件无法缓存时，移除缓存
						expired = append(expired, fname)
					case _canupdate, _cancache:
						// 本地文件可被缓存时，更新缓存
						m.update(cfile)
					}
					continue
				}

				switch status {
				case _notexist:
					// 本地文件被移除后，标记不存在
					cfile.exist = false
					cfile.bytes = nil
					cfile.info = nil
					continue

				case _notallowed, _willoverflow:
					// 本地文件不可缓存时，强制移除缓存
					mustExpired = append(mustExpired, fname)
					continue

				case _cancache, _canupdate:
					if info.ModTime().Equal(cfile.info.ModTime()) {
						// 本地文件未更新时缓存不变
						continue
					}
					// 本地文件已更新时更新缓存
					m.update(cfile)
					continue
				}
			}
			// 移除过期缓存
			m.RUnlock()
			if len(expired) > 0 {
				m.Lock()
				for _, fname := range mustExpired {
					m.usedSize -= m.filemap[fname].Size()
					delete(m.filemap, fname)
				}
				for _, fname := range expired {
					if m.filemap[fname].last.Add(m.gc).Before(time.Now()) {
						m.usedSize -= m.filemap[fname].Size()
						delete(m.filemap, fname)
					}
				}
				m.Unlock()
			}
		}
	})
}

func (m *MemoryCache) check(fname string) (os.FileInfo, int) {
	info, err := os.Stat(fname)
	if err != nil {
		if os.IsNotExist(err) {
			return info, _notexist
		}
		return info, _unknown
	}
	if info.Size() > m.singleFileAllow {
		return info, _notallowed
	}
	cfile, ok := m.filemap[fname]
	var currSize int64 = 0
	if ok {
		currSize = cfile.Size()
	}
	if m.usedSize-currSize+info.Size() > m.maxCap {
		return info, _willoverflow
	}
	if m.usedSize+info.Size() <= m.maxCap {
		return info, _cancache
	}
	return info, _canupdate
}

func (m *MemoryCache) update(c *Cachefile) {
	oldsize := c.Size()
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
	c.last = time.Now()

	m.usedSize = m.usedSize - oldsize + c.Size()
}
