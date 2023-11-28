package geecache

import (
	"fmt"
	"log"
	"sync"
)

type Getter interface {
	Get(key string) ([]byte, error)
}

// 定义了 GetterFunc 别名的目的是为了方便将一个函数类型适配到满足 Getter 接口的需求上
type GetterFunc func(key string) ([]byte, error)

// 当实际使用时，GetterFunc就是创建NewGroup时，
// 传入的用户自定义的函数GetterFunc(func(...)...) [需要将func定义一个别名]
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
	mu sync.RWMutex
	// 如何存放多个Group对象，采用map的形式
	groups = make(map[string]*Group)
)

// 创建一个Group实例
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	// 排他锁，只能允许同一时间单个进程只能创建一个
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

func GetGroup(name string) *Group {
	// 需要加锁，Rlock是共享锁，允许多个经常同时读取，但不允许写操作
	// 它不会监控是否有读操作，只是确保了此时没有进程能够获得 *写的锁*
	mu.RLock()
	defer mu.RUnlock()
	return groups[name]
}

// 定义Group的方法
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	mu.RLock()
	// 如果在mainCache中有缓存值，则返回缓存值
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}
	// 当mainCache中不存在时，调用load方法
	return g.load(key)
}

// load 调用local方法这个函数会调用回调函数获取源数据，并将源数据添加到mainCache中
func (g *Group) load(key string) (value ByteView, err error) {
	return g.getLocally(key)
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err

	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
