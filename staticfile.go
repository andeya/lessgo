package lessgo

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/lessgo/lessgo/utils"
)

func staticRoute() {
	// 注册固定的静态文件与目录
	DefLessgo.Echo.Static("/uploads", strings.ToLower(UPLOADS_DIR))
	DefLessgo.Echo.Static("/static", strings.ToLower(STATIC_DIR))
	DefLessgo.Echo.File("/favicon.ico", strings.ToLower(IMG_DIR)+"/favicon.ico")

	// 注册模块中的静态目录
	staticModule(BUSINESS_DIR)
	staticModule(SYSTEM_DIR)
}

func staticModule(root string) {
	dirs := utils.WalkDirs(root, VIEW_PKG)
	for _, dir := range dirs {
		DefLessgo.Echo.Static(urlPrefix(dir), dir)
	}
}

func urlPrefix(dir string) string {
	urlprefix := strings.TrimSuffix(dir, VIEW_PKG)
	urlprefix = strings.Replace(urlprefix, MODULE_SUFFIX, "", -1)
	return "/" + strings.ToLower(urlprefix)
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
		*bytes.Reader
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
	r := bytes.NewReader(bufferWriter.Bytes())
	c.exist = true
	c.info = info
	c.Reader = r
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
						cfile.Reader = nil
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
		return cfile.Reader, cfile.info, cfile.exist
	}

	m.Lock()
	defer m.Unlock()

	cfile, ok = m.filemap[fname]
	if ok {
		return cfile.Reader, cfile.info, cfile.exist
	}

	file, err := os.Open(fname)
	if err != nil {
		return nil, nil, false
	}
	defer file.Close()
	info, _ := file.Stat()
	var bufferWriter bytes.Buffer
	io.Copy(&bufferWriter, file)
	r := bytes.NewReader(bufferWriter.Bytes())
	cfile = &Cachefile{
		fname:  fname,
		Reader: r,
		info:   info,
		exist:  true,
	}
	m.filemap[fname] = cfile

	return cfile.Reader, cfile.info, cfile.exist
}
