package geecache

import (
	"sync"
)

type Getter interface {
	Get(key string) ([]byte, error)
}

// 定义了 GetterFunc 别名的目的是为了方便将一个函数类型适配到满足 Getter 接口的需求上
type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// 封装的带锁的cache不需要对外暴露，利用分组group来公开对外访问的方法
// 从而可以间接地访问缓存对象
type Group struct {
	name      string
	getter    Getter
	mainCache cache
}

var (
	// 多个Group对象，可能涉及多线程并发读写groups，所以需要用到读写锁
	mu     sync.RWMutex
	// 如何存放多个Group对象，采用map的形式
	groups = make(map[string]*Group)
)

// 创建一个Group实例
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
	}
	groups[name] = g
	return g
}

