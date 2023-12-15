package main

import (
	geerpc "codec"
	"codec/codec"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
)

func startServer(addr chan string) {
	// 创建一个TCP监听器，":0"是让系统自动分配一个可用端口
	// 返回一个监听器l和可能的错误
	l, err := net.Listen("tcp", ":0")

	if err != nil {
		log.Fatal("network error:", err)
	}
	// 打印监听器地址
	log.Println("start rpc server on", l.Addr())
	// 将地址转化为字符串，发送到通道中
	addr <- l.Addr().String()
	geerpc.Accept(l)
}

func main() {
	// 使用信道addr，确保服务端端口监听成功，客户端再发请求
	// 多个协程之间通过通道来传递数据
	addr := make(chan string)
	// 开启协程，不影响main程序的继续运行
	go startServer(addr)

	conn, _ := net.Dial("tcp", <-addr)
	defer func() { _ = conn.Close() }()
	time.Sleep(time.Second)
	// json.NewEncoder(conn)创建了一个编码器，将geerpc.DefaultOption这个结构体编码成json格式
	// 编码完成后的json数据会被发送到conn这个连接上
	_ = json.NewEncoder(conn).Encode(geerpc.DefaultOption)
	// conn中有read、write、Close函数隐式满足了io.ReadWriteCloser,所以可以作为变量加入
	// cc实际上是一个GobCodec的变量
	cc := codec.NewGobCodec(conn)

	// 模拟5次rpc过程
	for i := 0; i < 5; i++ {
		// 定义一个header
		h := &codec.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		// 调用Write、ReadHeader、ReadBody函数
		_ = cc.Write(h, fmt.Sprintf("geerpc req %d", h.Seq))
		_ = cc.ReadHeader(h)
		var reply string
		_ = cc.ReadBody(&reply)
		log.Println("reply:", reply)
	}
}
