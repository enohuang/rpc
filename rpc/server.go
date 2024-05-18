package rpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"reflect"
	"rpc/rpc/message"
	"rpc/rpc/serialize"
	"rpc/rpc/serialize/json"
	"strconv"
	"time"
)

type Server struct {
	services  map[string]reflectionStub
	serialize map[uint8]serialize.Serializer
}

func NewServer() *Server {
	res := &Server{
		services:  make(map[string]reflectionStub, 10),
		serialize: map[uint8]serialize.Serializer{},
	}
	res.RegisterSerializer(&json.Serializer{})
	return res
}

func (s *Server) RegisterSerializer(sl serialize.Serializer) {
	s.serialize[sl.Code()] = sl
}

func (s *Server) RegisterService(service Service) {
	s.services[service.Name()] = reflectionStub{
		s:          service,
		value:      reflect.ValueOf(service),
		serializer: s.serialize,
	}
}

func (s *Server) Start(network, addr string) error {
	listener, err := net.Listen(network, addr)
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

// 我们可以认为，一个请求包含两个部分
// 1. 长度字段， 用8个字节表示
// 2. 请求数据
// 响应也是这个规范
func (s *Server) handleConn(con net.Conn) error {
	for {

		reqBs, err := ReadMsg(con)
		if err != nil {
			return nil
		}

		req := message.DecodeReq(reqBs)

		ctx := context.Background()
		cancel := func() {}
		if deadlineStr, ok := req.Meta["deadline"]; ok {
			if deadline, er := strconv.ParseInt(deadlineStr, 10, 64); er == nil {
				ctx, cancel = context.WithDeadline(ctx, time.UnixMilli(deadline))

			}

		}
		oneway, ok := req.Meta["one-way"]
		if ok && oneway == "true" {
			ctx = CtxWithOneWay(ctx)
		}

		//不能直接用context.Background
		resp, err := s.Invoke(ctx, req)
		cancel()
		if err != nil {
			//处理业务 error
			resp.Error = []byte(err.Error())
			//return nil
		}

		resp.CalculateHeaderLength()
		resp.CalculateBodyLength()

		//res := message.EncodeResp(resp)
		_, err = con.Write(message.EncodeResp(resp))
		if err != nil {
			return err
		}

	}
}

// 这个必须返回response
func (s *Server) Invoke(ctx context.Context, req *message.Request) (*message.Response, error) {

	service, ok := s.services[req.ServiceName]

	resp := &message.Response{
		RequestID:  req.RequestID,
		Version:    req.Version,
		Compresser: req.Compresser,
		Serializer: req.Serializer,
	}

	if !ok {
		return resp, errors.New("你调用的服务不存在")
	}

	//直接开一个goroutine 让服务端不用等
	if IsOneWay(ctx) {
		go func() {
			_, _ = service.invoke(ctx, req)
		}()
		return nil, errors.New("micro: 微服务服务端 oneway 请求")
	}

	respData, err := service.invoke(ctx, req)
	resp.Data = respData
	/*	if err != nil {
		return resp, err
	}*/
	fmt.Println("(s *Server) Invoke", string(resp.Data), err)
	return resp, err
}

type reflectionStub struct {
	s          Service
	value      reflect.Value
	serializer map[uint8]serialize.Serializer
}

func (s *reflectionStub) invoke(ctx context.Context, req *message.Request) ([]byte, error) {
	//val := reflect.ValueOf(service)
	method := s.value.MethodByName(req.MethodName)

	in := make([]reflect.Value, 2)
	//暂时不知道怎么传
	in[0] = reflect.ValueOf(ctx)

	inReq := reflect.New(method.Type().In(1).Elem())
	serializer, ok := s.serializer[req.Serializer]
	if !ok {
		return nil, errors.New("不支持序列化协议")
	}
	err := serializer.Decode(req.Data, inReq.Interface())

	if err != nil {
		return nil, err
	}
	in[1] = inReq //reflect.ValueOf(inReq)
	results := method.Call(in)
	//results[0] 是返回值  [1] 是error

	if results[1].Interface() != nil {
		err = results[1].Interface().(error)
	}

	var res []byte
	if results[0].IsNil() {
		return nil, err
	} else {
		var er error
		res, er = serializer.Encode(results[0].Interface())
		if er != nil {
			return nil, er
		}
	}

	return res, err

}
