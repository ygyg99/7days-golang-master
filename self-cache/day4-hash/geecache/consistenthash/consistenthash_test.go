package consistenthash

import (
	"strconv"
	"testing"
)

func TestHashing(t *testing.T){
	// 创建一个新的实例,虚拟节点为3，Hash函数为简单的将输入的[]byte转为int
	hash := New(3,Hash(func(b []byte) uint32 {
		i,_ := strconv.Atoi(string(b))
		return uint32(i)
	}))
	// 添加真实节点(机器)[6,4,2]->虚拟节点[2,4,6,12,14,16,22,24,26]
	hash.Add("6","4", "2")

	// 添加测试用例
	testCases := map[string]string{
		"2":"2",
		"11":"2",
		"23":"4",
		"27":"2",
	}

	// 在testCases中依次测试应当存入哪一个DB(真实节点)
	for k,v := range testCases{
		if hash.Get(k) != v{
			t.Fatalf("Data: %s should be stored in DB: %s",k,v)
		}
	}

	// 添加一个真实节点[8]->虚拟节点[8,18,28]
	hash.Add("8")
	// testCases中"27"应该改为对应8
	testCases["27"] = "8"

	// 继续遍历进行测试
	for k,v := range testCases{
		if hash.Get(k) != v{
			t.Fatalf("Data: %s should be stored in DB: %s",k,v)
		}
	}
}
