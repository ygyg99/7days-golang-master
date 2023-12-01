package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// 定义函数别名
type Hash func([]byte) uint32

type Map struct {
	// Hash函数：将bytes转化为int(留下一个接口运行自定义Hash函数)
	hash Hash
	// 虚拟节点倍数
	repllicas int
	// 哈希环
	keys []int
	// 虚拟节点与真正节点映射表，键是虚拟节点hash值，值是真实节点名称
	hashmap map[int]string
}

// 定义初始化方法
func New(replicas int, fn Hash) *Map {
	// 初始化
	m := &Map{
		hash:      fn,
		repllicas: replicas,
		hashmap:   make(map[int]string),
	}
	// 当没有自定义Hash时则采用默认的
	if fn == nil {
		// 这里别在函数后面加(),直接是func赋值
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add 添加真实节点
func (m *Map) Add(keys ...string) {
	// 遍历每个key
	for _, key := range keys {
		// 每一个key都添加设定数量的虚拟节点
		for i := 0; i < m.repllicas; i++ {
			// 将"i+key"转化为hash值(int)
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			// 在hash环中添加节点值(int)
			m.keys = append(m.keys, hash)
			// 在节点映射表中添加映射
			m.hashmap[hash] = key
		}
	}
	// 将hash环重新sort成顺序的
	sort.Ints(m.keys)
}


// Get 查找输入key时，应当存入的数据库名称
func(m *Map)Get(key string)string{
	/* 自己的写法
	hash := m.hash([]byte(key))
	idx := 0
	for id,val := range m.keys{
		if val > int(hash){
			idx = id
			break
		}
	}
	if idx == 0{
		idx++
	}
	return m.hashmap[m.keys[idx-1]]
	*/
	lg := len(m.keys)
	if lg==0{
		return ""
	}
	hash := int(m.hash([]byte(key)))
	idx := sort.Search(lg,func(i int) bool {
		return m.keys[i] >= hash
	})
	// 如果没找到会返回lg，常规返回值是[0,lg)
	return m.hashmap[m.keys[idx%lg]]

}

