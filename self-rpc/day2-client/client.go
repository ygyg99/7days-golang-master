package client

import (
	"client/codec"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

// 定义常见错误
var (
	ErrShutdown = errors.New("connection is shut down")
)

// 封装结构体Call来承载一次RPC调用所需的信息
type Call struct {
	Seq           uint64 //表示RPC调用的序列号
	ServiceMethod string //表示被调用的服务及方法信息 format "<service>.<method>"
	// 使用inteface保证了接受和返回数据的灵活性
	Args  interface{} //函数的参数
	Reply interface{} //远程函数的返回值
	Error error
	// 支持异步调用，调用结束后会通知调用方
	Done chan *Call
}

func (call *Call) done() {
	call.Done <- call
}

type Client struct {
	// 具有ReadHeader、ReadBody、Write的结构体(可以进行解析)
	cc       codec.Codec      //(服务端)消息编解码结构
	opt      *Option          //选型参数
	sending  sync.Mutex       //(服务端)互斥锁，保证请求有序发送，防止报文混淆
	header   codec.Header     //请求头，请求发送是互斥的，每个客户端只需一个
	mu       sync.Mutex       //保护下面的参数
	seq      uint64           //给发送的请求编号，每个请求有唯一编号
	pending  map[uint64]*Call //存储未处理完的请求，key是编号，val是Call实例
	closing  bool             //用户的call是否结束(用户主动关闭)
	shutdown bool             //服务器是否叫停调用(遇错时才会被动关闭)
}

// 做类型检测，是否实现了close的方法
var _ io.Closer = (*Client)(nil)

// Close 关闭连接
func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing {
		return ErrShutdown
	}
	client.closing = true
	return client.cc.Close()
}

// IsAvailable判断是否客户端可用
func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	log.Println("1")

	return !client.shutdown && !client.closing

}

// 完善创建Client的功能
// 需要完成协议交换(发送Option消息给服务端)，协商消息的编解码方式
// ，再创建子协程调用receive接受响应
func NewClient(conn net.Conn, opt *Option) (*Client, error) {

	// 调用这个函数会返回一个用于实例化Codec的函数
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		err := fmt.Errorf("invalid codec type %s", opt.CodecType)
		log.Println("rpc client: codec error:", err)
		return nil, err
	}

	// 将options参数序列化送往服务端
	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Panicln("rpc client: options error:", err)
		_ = conn.Close()
		return nil, err
	}
	return newClientCodec(f(conn), opt), nil
}

// 创建Client， 接受Codec和Options参数填入client中
func newClientCodec(cc codec.Codec, opt *Option) *Client {
	client := &Client{
		seq:     1,
		cc:      cc,
		opt:     opt,
		pending: make(map[uint64]*Call),
	}
	go client.receive()
	return client
}

// Dial函数的实现，便于用户传入服务端地址(将Option作为可选参数)
func parseOptions(opts ...*Option) (*Option, error) {
	// 如果opts是空或者传入一个nil的参数
	if len(opts) == 0 || opts[0] == nil {
		return DefaultOption, nil
	}
	if len(opts) != 1 {
		return nil, errors.New("number of options is more than 1")
	}
	opt := opts[0]
	opt.MagicNumber = DefaultOption.MagicNumber
	if opt.CodecType == "" {
		opt.CodecType = DefaultOption.CodecType
	}
	return opt, nil
}

// 通过Dial函数，获得相应地址连接网络的conn并且进行参数的解析
func Dial(network, address string, opts ...*Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}
	// 通过网址获得连接
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	defer func() {
		if client == nil {
			_ = conn.Close()
		}
	}()
	return NewClient(conn, opt)
}

// 实现与Call相关的三个方法

// registerCall 注册一个远程调用请求Call并添加到pending中
func (client *Client) registerCall(call *Call) (uint64, error) {
	// 加锁，防止报文错乱

	client.mu.Lock()
	defer client.mu.Unlock()
	// 判断客户端是否可用,这里不能用IsAva函数，因为前面已经加了锁了，IsAva中也有锁，会造成死锁
	if client.closing || client.shutdown {
		return 0, ErrShutdown
	}

	// 填入Call的序号并且将候选列pending map中加入call
	call.Seq = client.seq
	client.pending[call.Seq] = call
	// call序号seq++
	client.seq++
	// 返回Call的序号

	return call.Seq, nil
}

// removeCall 根据seq从pending中移除对应的call,
func (client *Client) removeCall(seq uint64) *Call {
	// 加锁
	client.mu.Lock()
	defer client.mu.Unlock()

	// 在pending中查询(不用判断是否存在)
	call := client.pending[seq]
	delete(client.pending, seq)
	return call
}

// terminateCalls Clien/Server 发送错误时调用，设置shutdown，
// 并将err更新到pending待处理的call中
func (client *Client) terminateCalls(err error) {
	client.sending.Lock()
	client.mu.Lock()
	// func(){}定义了一个函数，后面跟的()表示函数的立即执行
	defer func() {
		client.sending.Unlock()
		client.mu.Unlock()
	}()
	client.shutdown = true
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}

// 实现客户端的两个重要功能，实现接收响应，发送请求

// receive 接收请求
func (client *Client) receive() {
	// 定义一个err，用于接收在整个过程中出现的错误err
	var err error
	for err == nil {
		// 读取请求头,获取请求Call的序号
		var h codec.Header
		if err = client.cc.ReadHeader(&h); err != nil {
			break
		}
		call := client.removeCall(h.Seq)
		switch {
		// call不存在时，直接readbody一个nil
		case call == nil:
			continue
		// call存在，但Error不为空
		case h.Error != "":
			call.Error = fmt.Errorf(h.Error)
			call.done()
		// 正常处理情况
		default:
			err = client.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body" + err.Error())
			}
			call.done()
		}
	}
	// 若有错误，直接调用终止
	client.terminateCalls(err)
}

// 发送请求
func (client *Client) send(call *Call) {
	// 保证发送请求的完整性
	client.sending.Lock()
	defer client.sending.Unlock()

	// 注册请求

	seq, err := client.registerCall(call)
	if err != nil {
		call.Error = err
		// 把这个call送入通道chan中
		call.done()
		return
	}

	// 准备请求头
	client.header.ServiceMethod = call.ServiceMethod
	client.header.Seq = seq
	client.header.Error = ""

	// 加密并且发送请求
	if err := client.cc.Write(&client.header, call.Args); err != nil {
		call := client.removeCall(seq)
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

// Go和Call是客户端暴露给用户的两个RPC服务调用接口

// Go是一个异步接口，返回call实例
func (client *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Panic("rpc client: done chan is unbuffered")
	}
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}

	client.send(call)
	return call
}

// Call是对Go的封装，阻塞call.Done，等待响应的返回，是一个同步接口
func (client *Client) Call(serviceMethod string, args, reply interface{}) error {

	call := <-client.Go(serviceMethod, args, reply, make(chan *Call, 1)).Done
	return call.Error
}
