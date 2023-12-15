package main

import (
	"client"
	geerpc "codec"
	"fmt"
	"log"
	"net"
	"sync"
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

	// Aceept函数是自己定义的server中的函数，相当于在原来的Accept函数上嵌套了一层，并且可以调用自己的解析方式
	geerpc.Accept(l)
}

func main() {
	log.SetFlags(0)
	// 使用信道addr，确保服务端端口监听成功，客户端再发请求
	// 多个协程之间通过通道来传递数据
	addr := make(chan string)
	// 开启协程，不影响main程序的继续运行
	go startServer(addr)

	// 将rpc调用改成使用client的方式，并且将上面创建的服务端的地址与client进行连接
	client, _ := client.Dial("tcp", <-addr)
	defer client.Close()
	time.Sleep(time.Second)

	var wg sync.WaitGroup

	// 模拟5次rpc过程,客户端，主要负责将请求发出去
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// log.Println("start")
			args := fmt.Sprintf("geerpc req %d", i)
			var reply string
			if err := client.Call("Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Println(1)
			log.Println("reply:", reply)
		}(i)
	}
	wg.Wait()
}
