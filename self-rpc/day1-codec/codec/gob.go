package codec

import (
	"bufio"
	"encoding/gob"
	"io"
	"log"
	"strconv"
)

// 首先定义了2种Codec，这个结构体由四部分构成，conn由构建函数传入。
// dec和enc对应gob的Decoder和Encoder，buf是为了防止阻塞而创建的带缓存的Writer
type GobCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer
	dec  *gob.Decoder
	enc  *gob.Encoder
}

var _ Codec = (*GobCodec)(nil)

func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}

// 实现 ReadHeader、ReadBody、Write 和 Close 方法
func (c *GobCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}

func (c *GobCodec) ReadBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c *GobCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()
	if err := c.enc.Encode(h); err != nil {
		log.Println("rpc codec: gob error encoding header:", err)
		return err
	}
	if err := c.enc.Encode(body); err != nil {
		log.Println("rpc codec: gob error encoding body:", err)
		return err
	}
	log.Println("&{", h.ServiceMethod, strconv.FormatUint(h.Seq,10), "}", body.(string))

	return nil
}
func (c *GobCodec) Close() error {
	return c.conn.Close()
}
