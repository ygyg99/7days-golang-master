package geecache

import (
	"continue_hash/lru"
	"sync"
)

// 一个大的cache包含有之前实现的lru存储和一个锁，以及最大存储所占大小
type cache struct {
	mu sync.Mutex
	lru *lru.Cache
	cacheBytes int64
}

// 定义get行为
func (c *cache) get(key string) (value ByteView, ok bool){
	// 先上锁
	c.mu.Lock()
	// 定义defer操作
	defer c.mu.Unlock()
	if c.lru == nil{
		return
	}
	if val,ook := c.lru.Get(key); ook{
		value, ok = val.(ByteView), ook
	}
	return 
}

// 定义add行为
func (c *cache) add(key string, value ByteView){
	c.mu.Lock()
	defer c.mu.Unlock()
	// 这点没考虑到，当进行结构体嵌套时，需要注意是否内部的结构体是否初始化
	if c.lru == nil{
		c.lru = lru.Init(c.cacheBytes,nil)
	}
	c.lru.Add(key,value)
}