package geecache

// 抽象两个接口，PeerPicker的PickPeer()方法用于根据传入的key选择对应的节点PeerGetter
// 当需要在其他节点查询key时，封装一个该功能的接口
type PeerPicker interface{
	PickPeer(key string)(peer PeerGetter, ok bool)
}
// PeerGetter的Get()方法用于从对应group查找缓存值【对应于HTTP的客户端】
// 定义一个接口用于通过网络请求返回缓存结果
type PeerGetter interface{
	Get(group string, key string)([]byte, error)
}