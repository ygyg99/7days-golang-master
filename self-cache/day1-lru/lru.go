package main

import (
	"container/list"
)

//键值对 entry 是双向链表节点的数据类型，
//在链表中仍保存每个值对应的 key 的好处在于，
//淘汰队首节点时，需要用 key 从字典中删除对应的映射

// 定义element的value是entry类型
type entry struct {
	key   string
	value Value
}

// 定义一个总的接口方法，对于不同的类型可能有不同的计算长度的方法
// Value use Len to count how many bytes it takes
type Value interface {
	Len() int
}

type Cache struct {
	maxBytes  int64      //最大存储容量
	nBytes    int64      //已占用存储
	ll        *list.List //双向链表
	cahce     map[string]*list.Element
	OnEvicted func(key string, value Value)
}

// Init
func Init(maxBytes int64, OnEvicted func(key string, value Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		nBytes:    0,
		ll:        list.New(),
		cahce:     make(map[string]*list.Element),
		OnEvicted: OnEvicted,
	}
}

func (c *Cache) Get(key string) (value Value, ok bool) {
	// 即使我们知道Value的any类型是*entry，但是不能直接调用*entry.value，
	//需要对any进行类型断言之后才能调用,且结构体传递指针比传其本身更有效率
	if ele, ok := c.cahce[key]; ok {
		value = ele.Value.(*entry).value
	}
	return
}

func (c *Cache) RemoveOldest() {
	// 对双向链表操作
	ele := c.ll.Back()
	c.ll.Remove(ele)
	// Value 存储的是entry，但也可以试试类型转换一下，如何出错了就报错
	entry := ele.Value.(*entry)
	// 对map操作
	delete(c.cahce, entry.key)
	// 更改c的容量(减去两部分的容量，一个是element中记录的entry的大小(双向链表)，还有一个是map中记录的key的大小)
	c.nBytes -= (int64(entry.value.Len()) + int64(len(entry.key)))
	// 调用回调函数
	c.OnEvicted(entry.key, entry.value)
}

func (c *Cache) Add(key string, value Value) {
	// 如果key已存在
	if ele, ok := c.cahce[key]; ok {
		/* v1.不用修改值，因为后面会将这个ele消除，加入一个新的
		// 修改value值
		ele.Value.(*entry).value = value
		*/

		/* v2.也不用这样，因为ll提供将目标移到最前端的操作
		// 将ele从双向链表中提到前面去
		c.ll.Remove(ele)
		c.ll.PushFront(&entry{key,value})
		// 将这个新的ele更新到cache中
		c.cahce[key] = c.ll.Front()
		*/

		/* v3.忘记修改内存了，不同的value的内存占有不同
		// 修改ele值，并将ele移到最前面()
		ele.Value.(*entry).value = value
		c.ll.MoveToFront(ele)
		*/

		// 修改双向链表
		c.ll.MoveToFront(ele)

		// pre_ele := ele  这里不应该直接赋值，因为指针赋值指向的同一个地址
		// key值没变，所以不用加减key的内存变化
		pre_bytes := ele.Value.(*entry).value.Len()
		// 更改值
		ele.Value.(*entry).value = value
		// 修改内存大小
		c.nBytes += int64(value.Len()) - int64(pre_bytes)
	} else {
		// 当key不存在时
		// 若容量满了
		if c.nBytes == c.maxBytes {
			// 读取需要删除的ele，将其从cache和ll中删除
			ele := c.ll.Back()
			c.ll.Remove(ele)
			delete(c.cahce, ele.Value.(*entry).key)
			// 计算容量变化
			en := ele.Value.(*entry)
			c.nBytes -= int64(en.value.Len()) + int64(len(en.key))
		}
		// 直接添加
		// 1、从ll首部添加 2、从cache添加 3、修改容量
		ele := c.ll.PushFront(&entry{key, value})
		c.cahce[key] = ele
		c.nBytes += int64(value.Len()) + int64(len(key))

		// 这个遗忘了，还需要在检测添加后是否溢出了
		for c.nBytes > c.maxBytes && c.maxBytes != 0 {
			c.RemoveOldest()
		}
		c.OnEvicted(key, value)
	}
}

func (c *Cache) Len() int {
	return c.ll.Len()
}
