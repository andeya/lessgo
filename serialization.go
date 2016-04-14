package lessgo

import (
	"fmt"
	"sync"
)

// 可序列化存储的虚拟路由信息
type SerialRouter struct {
	Id            string   // parent.id+VirtHandler.prefix+[METHOD1]+[METHOD...]
	Type          int      // 操作类型: 根目录/路由分组/操作
	Name          string   // 名称(建议唯一)
	Children      []string // 子节点id列表
	Enable        bool     // 是否启用当前路由节点
	Middleware    []string // 中间件 (允许运行时修改)
	VirtHandlerId string   // 虚拟操作id
}

var (
	serialRouterMap  = map[string]*SerialRouter{}
	serialRouterLock sync.RWMutex
)

// 注册序列化虚拟路由信息
func RegSerialRouter(s *SerialRouter) {
	serialRouterLock.Lock()
	defer serialRouterLock.Unlock()
	serialRouterMap[s.Id] = s
}

// 从根路径开始反序列化虚拟路由，返回根路由
func ToVirtRouter() (root *VirtRouter, err error) {
	serialRouterLock.RLock()
	defer serialRouterLock.RUnlock()
	var r *SerialRouter
	for _, v := range serialRouterMap {
		if v.Type == ROOT {
			r = v
			break
		}
	}
	if r == nil {
		return nil, fmt.Errorf(`无法找到根路由的信息`)
	}
	// 清空virtRouterMap
	cleanVirtRouterMap()
	// 反序列化到虚拟路由树
	root, err = r.virtRouterTree()
	if err != nil {
		return
	}
	addVirtRouter(root)
	return
}

// 返回序列化虚拟路由记录表的副本
func SerialRouterMap() map[string]*SerialRouter {
	serialRouterLock.RLock()
	defer serialRouterLock.RUnlock()
	srs := make(map[string]*SerialRouter, len(serialRouterMap))
	for k, v := range serialRouterMap {
		srs[k] = v
	}
	return srs
}

// 创建虚拟路由树
func (s *SerialRouter) virtRouterTree() (*VirtRouter, error) {
	vh, _ := GetVirtHandler(s.VirtHandlerId)
	vr := &VirtRouter{
		id:          s.Id,
		typ:         s.Type,
		name:        s.Name,
		children:    []*VirtRouter{},
		enable:      s.Enable,
		middleware:  s.Middleware,
		VirtHandler: vh,
	}
	for _, id := range s.Children {
		child, ok := getSerialRouterMap(id)
		if !ok {
			return vr, fmt.Errorf("不存在id为 %v 的虚拟路由节点", id)
		}
		cvr, err := child.virtRouterTree()
		if err != nil {
			return vr, err
		}
		cvr.SetParent(vr)
	}
	vr.reset()
	return vr, nil
}

func getSerialRouterMap(id string) (*SerialRouter, bool) {
	serialRouterLock.RLock()
	defer serialRouterLock.RUnlock()
	s, ok := serialRouterMap[id]
	return s, ok
}

func cleanSerialRouterMap() {
	serialRouterLock.Lock()
	defer serialRouterLock.Unlock()
	serialRouterMap = map[string]*SerialRouter{}
}
