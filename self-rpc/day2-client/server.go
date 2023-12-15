package client

import (
	"client/codec"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

const MagicNumber = 0x3bef5c

// 可以定义一些选型参数
type Option struct {
	MagicNumber int
	// 用于表示编解码器的类型
	CodecType codec.Type
}

// 定义一个默认参数选型
var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

// 首先定义结构体Server，无任何成员字段
type Server struct{}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

// 信息处理过程
// 反序列得到Option实例，检查Option内的参数，并将处理转给serverCodec
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() { _ = conn.Close() }()
	var opt Option

	// 

	// 先使用json的decoder反序列化得到Option实例,将http请求的参数填充如opt中
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	// 检查Option的MagicNumber和CodecType是否正确
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
		return
	}
	// 根据CodeType得到对应的消息编解码器，接下来处理交给serverCodec
	// f是Codec的构造函数
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server: invalid codec type %s", opt.CodecType)
		return
	}
	server.serveCodec(f(conn))
}

var invalidRequest = struct{}{}

// serverCodec 主要包含三个阶段1、读取请求 2、处理请求 3、回复请求
func (server *Server) serveCodec(cc codec.Codec) {
	// 新建一个锁用来保证每次发送的都是一个完整的信息
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	// 在一次连接中，允许接收多个请求，使用for来无限制的等待请求到来，直到发送错误
	for {
		req, err := server.readRequest(cc)
		if err != nil {
			// 尽力而为，只有在header解析失败时才终止循环
			if req == nil {
				break
			}
			req.h.Error = err.Error()
			// 处理请求可以并发，但是回复请求的报文需要逐个发送，使用sending来保证
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		// 使用了协程并发执行请求
		go server.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
	_ = cc.Close()
}

type request struct {
	h            *codec.Header
	argv, replyv reflect.Value
}

// readRequestHeader 用于解析报文头
func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	// 调用ReadHeader方法用于将报文解析存入h中
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Panicln("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	// 读取Header
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	// 将header数据存入一个新的req中
	req := &request{h: h}
	// 使用Go的reflect包根据空字符串""的类型创建一个新的反射对象。
	// 这将创建一个指向字符串的指针(*string)，并将其赋给argv
	req.argv = reflect.New(reflect.TypeOf(""))
	// 将调用了reflect.value()的方法Interface()[利用接口的value封装一下]，并利用ReadBody读取
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err:", err)
	}
	return req, nil
}

func (server *Server)sendResponse(cc codec.Codec, h *codec.Header, body interface{},sending *sync.Mutex){
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil{
		log.Println("rpc server: write response err:", err)
	}
}

func(server *Server)handleRequest(cc codec.Codec,req *request, sending *sync.Mutex, wg *sync.WaitGroup){
	defer wg.Done()
	log.Println()
	// ValueOf 用于将字符串转化为reflect.value并
	req.replyv = reflect.ValueOf(fmt.Sprintf("geerpc resp %d", req.h.Seq))
	server.sendResponse(cc, req.h, req.replyv.Interface(),sending)
}

// 实现了Accept方式，net.Listener作为参数,for循环等待socket连接建立，并开启子协程处理
// 处理过程交给了ServerConn方法
func (server *Server) Accept(lis net.Listener) {
	// 持续接受连接
	for {
		// 这是一个阻塞操作，一直等待到有连接进来
		// 返回一个代表连接的对象和可能的错误
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		// 创建一个协程用于处理刚刚接受的连接
		go server.ServeConn(conn)
	}
}

func Accept(lis net.Listener) { DefaultServer.Accept(lis) }
