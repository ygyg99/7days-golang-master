package main

import (
	"flag"
	"fmt"
	"log"
	geecache "multinodes"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createGroup() *geecache.Group {
	return geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

// startCacheServer() 用来启动缓存服务器：创建 HTTPPool，添加节点信息，注册到 gee 中，
// 启动 HTTP 服务（共3个端口，8001/8002/8003），用户不感知。
// addr是HTTPPool的self地址，addrs是需要注册的所有地址，即每个HTTPPool对应于一个真实节点
func startCacheServer(addr string, addrs []string, gee *geecache.Group) {
	// 利用addr来注册一个HTTPPool
	peers := geecache.NewHTTPPool(addr)
	// 在这个Pool中注册所有的真实节点
	peers.Set(addrs...)
	// 将这个Pool注册到group中
	gee.RegisterPeers(peers)
	log.Println("geecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

// 用于启动一个API服务(端口：9999)用于与用户交互、感知
func startAPIServer(apiAddr string, gee *geecache.Group) {
	http.Handle("/api",http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			// 获取key的信息，并将其打印出来
			view,err := gee.Get(key)
			if err != nil{
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:],nil))
}
func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8081, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001:"http://localhost:8001",
		8002:"http://localhost:8002",
		8003:"http://localhost:8003",
	}

	var addrs []string
	for _,v := range addrMap{
		addrs = append(addrs, v)
	}
	gee := createGroup()
	if api{
		go startAPIServer(apiAddr, gee)
	}
	startCacheServer(addrMap[port],[]string(addrs),gee)
}