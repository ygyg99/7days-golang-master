// byteview 结构体（存入了key-value，利用一个只读结构来特别操作这些存入的value）
// 主要用于封装存入的value（字节数组），以及操作字节数组的一些操作（如获取长度）
package geecache

type ByteView struct {
	b []byte
}

func (v ByteView) Len() int {
	return len(v.b)
}

// 返回一个byte数据的备份
func (v ByteView) ByteSlice() []byte {
	copy := make([]byte, v.Len())
	for idx,b := range v.b{
		copy[idx] = b
	}
	return copy
}

// 将byte数组整合为一个string
func (v ByteView) String() string{
	return string(v.b)
}
