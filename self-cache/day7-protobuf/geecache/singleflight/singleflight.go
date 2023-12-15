package singleflight

import "sync"
// call 代表正在进行中/已经结束的请求 
type call struct{
	// 使用sync.WaitGroup()避免重入
	wg sync.WaitGroup
	val interface{}
	err error
}

// singleflight主数据结构 管理不同key的请求
type Group struct{
	// 保护m的锁
	mu sync.Mutex
	m map[string]*call
}


// Do方法 针对相同的key，无论Do被调用多少次，函数fn只会被调用一次
// g.mu 作用是保护Group的成员变量m不被并发读写
func (g *Group)Do(key string, fn func()(interface{},error))(interface{},error){
	g.mu.Lock()
	// 当map为空时需要定义一个新的(每当有嵌套结构体时)
	if g.m == nil{
		g.m = make(map[string]*call)
	}
	// 如果在map中存的有值，则返回已经记录的值
	if c,ok := g.m[key];ok{
		g.mu.Unlock()
		// 如果请求正在进行中，则等待
		c.wg.Wait()
		// 请求结束返回结果
		return c.val, c.err
	}
	// 已有数据中没有则新建一个
	c := new(call)
	// 发请求前提前加锁
	c.wg.Add(1)
	// 添加到g.m中，表明key已经有对应的请求在处理了
	g.m[key] = c
	g.mu.Unlock()

	// 调用fn发起请求
	c.val, c.err = fn()
	// 请求结束(锁减一)
	c.wg.Done()

	g.mu.Lock()
	// 更新g.m
	// 不删除时，如果key对应值变化，所得值仍然为旧值，而且singleflight只是一个缓存器，不需要有存储功能
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err

}