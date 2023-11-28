package geecache

import (
	"fmt"
	"reflect"
	"testing"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestGetter(t *testing.T) {
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})

	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Errorf("callback failed")
	}
}

func TestGet(t *testing.T) {
	// 用一个map模拟耗时的数据库
	var db = map[string]string{
		"Tom":  "630",
		"Jack": "589",
		"Sam":  "567",
	}
	// 定义一个记录每个key调用回调函数的次数(按理一个key最多调用一次，若调用多次则说明，cache中没能按预期存入数据)
	loadCounts := make(map[string]int, len(db))
	// 定义一个查询成绩的group
	gee := NewGroup("scores", 2<<10, GetterFunc(func(key string) ([]byte, error) {
		t.Logf("[SlowDB] search key: %s", key)
		if v, ok := db[key]; ok {
			loadCounts[key]++
			return []byte(v), nil
		}
		return nil, fmt.Errorf("%s key not exist", key)
	}))
	// 遍历查询
	for k, v := range db {
		// 从缓存中查找，如果查找失败获取值不对则报错
		if val, ok := gee.Get(k); ok != nil || val.String() != v {
			t.Errorf("cache hit,but failed to get value of key: %s", k)
		}
		// 验证load是否最多一次(有个bug，就是当对其进行Remove时，没有将loadCounts清空)
		if _, ok := gee.Get(k); ok != nil || loadCounts[k] > 1 {
			t.Errorf("cache %s miss", k)
		}
	}
	// 验证查询一个不存在的事物
	if _, err := gee.Get("unkown"); err == nil {
		t.Fatalf("get wrong,should be none")
	}
}

func TestGetGroup(t *testing.T) {
	name := "scores"
	NewGroup(name, 2<<10, GetterFunc(
		func(key string) (bytes []byte, err error) { return }))
	if group := GetGroup(name); group == nil || group.name != name {
		t.Fatalf("group %s not exist", name)
	}
	if group := GetGroup("unkown"); group != nil {
		t.Fatalf("group %s not exist", name)
	}
}