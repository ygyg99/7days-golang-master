package geecache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

const defaultBasePath = "/_geecache"

type HTTPPool struct {
	// 包含有自身的独特路径self:主机名/IP和端口
	// 和基础公共路径basePath:作为节点间通讯地址的前缀
	self     string
	basePath string
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// v ...interface{}是一个可变参数(可传入任意数量的参数)

// Log 用于记录起始信息(这种函数信息记忆一下)
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 验证是否当前请求含有需要的前缀
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic(fmt.Sprintf("http prefix wrong: expect %s,has %s", p.basePath, r.URL.Path))
	}
	// 记录当前方法和路由
	p.Log("Method:%s, URL:%s", r.Method, r.URL.Path[len(p.basePath):])
	// 删去前缀
	url := r.URL.Path[len(p.basePath):]
	// 路径的格式应该为 /<basepath>/<groupname>/<key>
	// 将路由路径由/分割成不同的部分,当未得到两个时返回BadRequest
	parts := strings.SplitN(url, "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		// 不要忘了return
		return
	}
	// 读取groupname和key
	groupname, key := parts[0], parts[1]
	// 根据name获取group,当未获得时返回StatusNotFound
	group := GetGroup(groupname)
	if group == nil {
		http.Error(w, "no such group:"+groupname, http.StatusNotFound)
		return
	}
	// 从group中获取key的view，并进行失败判断返回StatusInternalServerErr
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, "no such key", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	// 利用Write自定义显示的信息
	w.Write(view.ByteSlice())
}
