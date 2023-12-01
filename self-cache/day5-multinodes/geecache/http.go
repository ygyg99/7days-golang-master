package geecache

import (
	"fmt"
	"io"
	"log"
	"multinodes/consistenthash"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_geecache"
	defaultReplicas = 50
)
// 每个HTTPPool都有自身独特的地址，其内部还存有该存到哪个节点的hash结构，以及对应于各个节点对应的映射表
type HTTPPool struct {
	// 包含有自身的独特路径self:主机名/IP和端口
	// 和基础公共路径basePath:作为节点间通讯地址的前缀
	self     string
	basePath string

	// 下面是新增的东西

	// 加锁,保证peers和httpGetters
	mu sync.Mutex
	// 可以实现一致hash的结构
	peers *consistenthash.Map
	// 映射key:远程节点地址 eg:"http://...",value:访问地址逻辑
	httpGetters map[string]*httpGetter
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
	url := r.URL.Path[len(p.basePath)+1:]
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

// 实现PeerPicker接口，用于将HTTPPool中的节点添加并完善映射map
func (p *HTTPPool) Set(peers ...string){
	p.mu.Lock()
	defer p.mu.Unlock()
	// 定义一个新的一致性Hash的peers，可以根据key来选择节点
	p.peers = consistenthash.New(defaultReplicas,nil)
	// 在peers中添加节点
	p.peers.Add(peers...)
	// 构建一个map，映射远程节点和对应的httpGetter(获取远程数据的接口)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	// 遍历peers，将其中每个节点对应到httpGetter完善到映射表中
	for _, peer := range peers{
		p.httpGetters[peer] = &httpGetter{baseURL: peer+p.basePath}
	}
}

// 通过key在Pool中看会搜索哪一个真实节点，再对地址进行判断是否需要返回一个远程获取的接口
func(p *HTTPPool)PickPeer(key string)(PeerGetter,bool){
	p.mu.Lock()
	defer p.mu.Unlock()
	// 从peer(一致性hash)中寻找key对应的节点地址
	if peer:=p.peers.Get(key);peer!=""&&peer!=p.self{
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer],true
	}
	// 如果是当前HTTPPool的地址则返回空
	return nil,false
}
var _ PeerPicker = (*HTTPPool)(nil)

// 定义一个结构体用于实现PeerGetter接口(属于客户端类，发送请求)
type httpGetter struct {
	baseURL string
}
// 分布实现，前面是实现了一个接口，后面是实现了具体的功能，从远程端口获取内容
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	// 利用Sprintf拼接字符串 baseURL+group+"/"+key
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	// 向u网址放松get请求并返回内容
	res, err := http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("url wrong: %v", u)
	}
	// 记得关闭响应主体，防止出现资源泄露
	defer res.Body.Close()
	// 如果返回状态码不是OK
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned:%v", res.Status)
	}
	// 将返回内容利用io.ReadAll函数转化为[]byte,并进行错误判断
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body error: %v", err)
	}
	return bytes, nil
}

// 保证实现了该接口
// 一个小技巧
var _ PeerGetter = (*httpGetter)(nil)
