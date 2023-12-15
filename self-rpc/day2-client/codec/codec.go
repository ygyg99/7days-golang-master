package codec

import "io"

type Header struct {
	// 服务名和方法名，通常与Go语言中的结构体和方法映射
	ServiceMethod string // format "Service.Method"
	// 请求的序号，用于区分不同的请求
	Seq uint64
	// 错误信息
	Error string
}

// 抽象出对消息体编解码的接口Codec[抽象出接口是为了实现不同的Codec实例]
type Codec interface {
	io.Closer
	// 一个方法签名，用于描述接口/结构体中的一个方法
	// ReadHeader->方法名称；*Header->接受参数；error->返回类型
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

// 抽象出Codec的构造函数
type NewCodecFunc func (io.ReadWriteCloser) Codec

// 可以通过Codec的Type得到构造函数，从而构建Codec实例
type Type string

// 定义了2种Codec，Gob和Json
const(
	GobType Type = "application/gob"
	JsonType Type = "application/json"
)

var NewCodecFuncMap map[Type]NewCodecFunc


// 当导入包的时候，init函数会自动被执行
func init(){
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}