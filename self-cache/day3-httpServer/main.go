package main

import (
	"fmt"
	geecache "http"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	// 这个函数用于启动一个 HTTP 服务器，监听本地 8081 端口，并使用 s 作为处理程序。
	// 这意味着当有请求到达时，会调用 s 的 ServeHTTP 方法来处理请求
	geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
	addr := "localhost:8081"
	peers := geecache.NewHTTPPool(addr)
	log.Println("geecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr, peers))
}
