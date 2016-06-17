package lessgo

import (
	"io/ioutil"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type MemoryCache struct {
	usedSize        int64                 // 已用容量
	singleFileAllow int64                 // 允许的最大文件
	maxCap          int64                 // 最大缓存总量
	enable          *bool                 // 是否启动缓存
	gc              time.Duration         // 缓存更新检查时长及动态过期时长
	filemap         map[string]*Cachefile // 已监控的文件缓存
	trigger         chan struct{}         // 主动触发扫描本地文件
	once            sync.Once
	sync.RWMutex
}

func NewMemoryCache(singleFileAllow, maxCap int64, gc time.Duration) *MemoryCache {
	return &MemoryCache{
		singleFileAllow: singleFileAllow,
		maxCap:          maxCap,
		gc:              gc,
		enable:          new(bool),
		filemap:         map[string]*Cachefile{},
		trigger:         make(chan struct{}),
	}
}

func (m *MemoryCache) Enable() bool {
	m.RLock()
	defer m.RUnlock()
	return *m.enable
}

func (m *MemoryCache) SetEnable(bl bool) {
	m.Lock()
	defer m.Unlock()
	_bl := *m.enable
	if _bl == bl {
		return
	}
	*m.enable = bl
	if !_bl { //没开启就开启
		m.once = sync.Once{}
		m.memoryCacheMonitor()
	}
}

// 返回文件字节流、文件信息、文件是否存在
func (m *MemoryCache) GetCacheFile(fname string) ([]byte, os.FileInfo, bool) {
	m.RLock()
	cfile, ok := m.filemap[fname]
	if ok {
		m.RUnlock()
		// 存在缓存直接输出
		return cfile.get()
	}
	m.RUnlock()

	m.Lock()
	defer m.Unlock()

	// 写锁成功后，再次检查缓存是否已存在，存在则输出
	cfile, ok = m.filemap[fname]
	if ok {
		return cfile.get()
	}

	// 读取本地文件
	file, err := os.Open(fname)
	if err != nil {
		return nil, nil, false
	}
	defer file.Close()
	info, _ := file.Stat()
	buf, err := ioutil.ReadAll(file)
	if err != nil {
		return buf, info, true
	}
	// 检查是否加入缓存
	if size := m.usedSize + info.Size(); size <= m.maxCap {
		m.filemap[fname] = &Cachefile{
			fname: fname,
			bytes: buf,
			info:  info,
			exist: true,
			time:  time.Now().Unix(),
		}
		atomic.StoreInt64(&m.usedSize, size)
	}
	return buf, info, true
}

// 主动触发扫描本地文件
func (m *MemoryCache) TriggerScan() {
	defer func() {
		// 当扫描刚刚已被触发时，trigger通道会被关闭重开，因此可能发生panic
		recover()
	}()
	m.trigger <- struct{}{}
}

// 全局监控协程
func (m *MemoryCache) memoryCacheMonitor() {
	enable := m.enable
	go m.once.Do(func() {
		defer func() {
			// 退出清理缓存
			m.filemap = make(map[string]*Cachefile)
		}()
		for *enable {
			// 屏蔽上次扫描期间，主动触发的不必要的扫描请求
			close(m.trigger)
			m.trigger = make(chan struct{})

			// 等待扫描本地文件
			select {
			case <-time.After(m.gc):
			// 定时更新
			case <-m.trigger:
				// 主动触发更新
			}

			// 开始执行扫描
			m.RLock()
			for _, cfile := range m.filemap {
				// 检查缓存超时，超时则加入过期列表
				if cfile.getTime().Add(m.gc).Before(time.Now()) {
					m.RUnlock()
					m.Lock()
					m.delete(cfile)
					m.Unlock()
					m.RLock()
					continue
				}

				// 获取是否更新文件的指示
				status := m.check(cfile)
				switch status {
				case _unknown, _nochange:
					// 本地文件状态未知或为改变时，保持现状
					continue

				case _notexist:
					// 本地文件被移除，则清空缓存并标记文件不存在
					cfile.clean()
					continue

				case _failupdate:
					// 不可更新时，清空并移除文件缓存
					m.RUnlock()
					m.Lock()
					m.delete(cfile)
					m.Unlock()
					m.RLock()
					continue

				case _canupdate:
					// 先清空缓存，再更新缓存
					m.update(cfile, false)
					continue

				case _preupdate:
					// 预加载的形式更新缓存
					m.update(cfile, true)
				}
			}
			m.RUnlock()
		}
	})
}

const (
	_unknown    = iota // 文件状态未知
	_notexist          // 文件不存在
	_nochange          // 文件未修改
	_failupdate        // 文件被修改，但无法更新
	_preupdate         // 文件被修改，可用预加载的形式更新
	_canupdate         // 文件被修改，可先删除后更新
)

// 扫描本地文件状态
func (m *MemoryCache) check(c *Cachefile) int {
	c.RLock()
	defer c.RUnlock()
	info, err := os.Stat(c.fname)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在
			return _notexist
		}
		// 文件状态未知
		return _unknown
	}

	if c.exist && c.info.ModTime().Equal(info.ModTime()) {
		// 文件未修改时不更新
		return _nochange
	}

	// 文件被修改后，或被标记为不存在的文件被重新发现

	if info.Size() > m.singleFileAllow {
		// 超出单个文件上限时不更新
		return _failupdate
	}
	currSize := int64(len(c.bytes))
	if m.usedSize-currSize+info.Size() > m.maxCap {
		// 剩余空间不足时不更新
		return _failupdate
	}
	if m.usedSize+info.Size() <= m.maxCap {
		// 可以预加载的形式更新
		return _preupdate
	}
	// 可以先删除后更新
	return _canupdate
}

// 更新缓存
func (m *MemoryCache) update(c *Cachefile, preupdate bool) {
	oldsize := c.size()
	defer func() {
		// 更新内存使用状态
		atomic.AddInt64(&m.usedSize, c.size()-oldsize)
	}()
	// 不可预加载时清空文件缓存，并写锁定
	if !preupdate {
		c.Lock()
		defer c.Unlock()
		c.bytes = nil
		c.info = nil
	}
	// 读取本地文件
	file, err := os.Open(c.fname)
	if err != nil {
		return
	}
	defer file.Close()
	info, _ := file.Stat()
	buf, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}
	if preupdate {
		c.Lock()
		defer c.Unlock()
	}
	// 写入缓存
	c.bytes = buf
	c.info = info
	c.exist = true
	c.time = time.Now().Unix()
}

// 删除文件缓存
func (m *MemoryCache) delete(c *Cachefile) {
	c.Lock()
	defer c.Unlock()
	delete(m.filemap, c.fname)
	m.usedSize -= int64(len(c.bytes))
	c.exist = false
	c.bytes = nil
	c.info = nil
}

type Cachefile struct {
	fname string      // 文件全名
	info  os.FileInfo // 文件信息
	bytes []byte      // 文件字节流
	time  int64       // 最近一次访问或更新时间，用于gc回收
	exist bool        // 文件在本地是否存在，避免每次扫描本地文件
	sync.RWMutex
}

// 获取缓存的文件大小
func (c *Cachefile) size() int64 {
	c.RLock()
	defer c.RUnlock()
	return int64(len(c.bytes))
}

// 获取缓存文件的内容、信息、存在性
func (c *Cachefile) get() ([]byte, os.FileInfo, bool) {
	c.RLock()
	defer c.RUnlock()
	atomic.StoreInt64(&c.time, time.Now().Unix())
	return c.bytes, c.info, c.exist
}

// 获取最近一次访问或更新时间
func (c *Cachefile) getTime() time.Time {
	c.RLock()
	defer c.RUnlock()
	return time.Unix(c.time, 0)
}

// 获取文件存在状态
func (c *Cachefile) getExist() bool {
	c.RLock()
	defer c.RUnlock()
	return c.exist
}

// 清空文件缓存并标记文件不存在
func (c *Cachefile) clean() {
	c.Lock()
	defer c.Unlock()
	c.exist = false
	c.bytes = nil
	c.info = nil
}
