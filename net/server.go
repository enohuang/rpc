package net

import (
	"errors"
	"net"
	"rpc/rpc"
)

func Serve(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		//端口被占用
		return err
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go func() {
			if err := handleConn(conn); err != nil {
				conn.Close()
			}
		}()
	}

	return nil
}

func handleConn(con net.Conn) error {
	for {
		bs := make([]byte, 8) //分配好内存

		//一般不用判断n
		n, err := con.Read(bs)
		//不可挽救
		/*if err == net.ErrClosed || err == io.EOF || err == io.ErrUnexpectedEOF{
			return err
		}

		if err != nil {
			continue
		}*/
		//建议出现错误，直接关闭
		if err != nil {
			return err
		}

		/*if n != 8 {
			return errors.New("没有读够数据")
		}*/

		res := handleMsg(bs)
		n, err = con.Write(res)
		//建议出现错误，直接关闭
		//不可挽救
		/*if err == net.ErrClosed || err == io.EOF || err == io.ErrUnexpectedEOF{
			return err
		}

		if err != nil {
			continue
		}*/
		if err != nil {
			return err
		}
		if n != len(res) {
			return errors.New("micro: 没写完数据")
		}
	}
}

func handleMsg(req []byte) []byte {
	res := make([]byte, 2*len(req))
	copy(res[:len(req)], req)
	copy(res[len(req):], req)
	return res
}

type Server struct {
	network string
	addr    string
}

func NewServer(network, addr string) *Server {
	return &Server{}
}

func (s *Server) Start(network, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		//端口被占用
		return err
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go func() {
			if err := s.handleConn(conn); err != nil {
				conn.Close()
			}
		}()
	}
}

const numOfLengthBytes = 8

// 我们可以认为，一个请求包含两个部分
// 1. 长度字段， 用8个字节表示
// 2. 请求数据
// 响应也是这个规范
func (s *Server) handleConn(con net.Conn) error {
	for {
		// lenBs 是长度字段的字节表示
		/*		lenBs := make([]byte, numOfLengthBytes) //分配好内存

				//一般不用判断n
				_, err := con.Read(lenBs)
				if err != nil {
					return err
				}
				//我消息有多长
				length := binary.BigEndian.Uint64(lenBs)
				reqBs := make([]byte, length)
				_, err = con.Read(reqBs)*/
		reqBs, err := rpc.ReadMsg(con)
		if err != nil {
			return err
		}

		respData := handleMsg(reqBs)
		/*	respLen := uint64(len(respData))

			res := make([]byte, respLen+numOfLengthBytes)

			binary.BigEndian.PutUint64(res[:numOfLengthBytes], respLen)
			copy(res[numOfLengthBytes:], respData)*/
		//res, _ := (respData)
		_, err = con.Write(respData)
		if err != nil {
			return err
		}

	}
}
