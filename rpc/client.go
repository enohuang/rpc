package rpc

import (
	"context"
	"errors"
	"github.com/silenceper/pool"
	"net"
	"reflect"
	"rpc/rpc/message"
	"rpc/rpc/serialize"
	json2 "rpc/rpc/serialize/json"
	"strconv"
	"time"
)

//mockgen -destination=micro/rpc/mock_proxy.gen.go -package=rpc  -source=micro/rpc/types.go Proxy

// InitClientProxy 替换掉 GetById 之类的的函数类型的字段赋值
func (c *Client) InitClientProxy(service Service) error {
	//在这里初始化一个 Proxy
	return setFuncField(service, c, c.serialize)
}

func setFuncField(service Service, p Proxy, s serialize.Serializer) error {

	if service == nil {
		return errors.New("rpc : 不支持 nil")
	}

	val := reflect.ValueOf(service)
	typ := val.Type()
	//只支持指向结构体的一级指针
	if typ.Kind() != reflect.Pointer || typ.Elem().Kind() != reflect.Struct {
		return errors.New("rpc : 只支持指向结构体的一级指针")
	}

	val = val.Elem()
	typ = typ.Elem()
	numField := typ.NumField()
	for i := 0; i < numField; i++ {
		fieldTyp := typ.Field(i)
		fieldVal := val.Field(i)

		if fieldVal.CanSet() {

			//这个地方才是真正的将本地调用铺抓到的地方
			//我要设置给 GetById
			fn := func(args []reflect.Value) (results []reflect.Value) {

				retVal := reflect.New(fieldTyp.Type.Out(0).Elem())

				ctx := args[0].Interface().(context.Context)
				//序列化只作用于协议体，不作用于协议头
				reqData, err := s.Encode(args[1].Interface())
				if err != nil {
					return []reflect.Value{retVal, reflect.ValueOf(err)}
				}

				meta := make(map[string]string, 2)

				if deadline, ok := ctx.Deadline(); ok {
					meta["deadline"] = strconv.FormatInt(deadline.UnixMilli(), 10)
				}

				if IsOneWay(ctx) {
					//meta = map[string]string{"one-way": "true"}
					meta["one-way"] = "true"
				}

				req := &message.Request{
					ServiceName: service.Name(),
					MethodName:  fieldTyp.Name,
					Data:        reqData,
					Serializer:  s.Code(),
					Meta:        meta,
				}

				req.CalculateHeaderLength()
				req.CalculateBodyLength()

				resp, err := p.Invoke(ctx, req)

				if err != nil {
					return []reflect.Value{retVal, reflect.ValueOf(err)}
				}

				var retErr error
				if len(resp.Error) > 0 {
					//服务端传过来的error
					retErr = errors.New(string(resp.Error))
				}

				if len(resp.Data) > 0 {
					err = s.Decode(resp.Data, retVal.Interface())
					if err != nil {
						//反序列化error
						return []reflect.Value{retVal, reflect.ValueOf(err)}
					}
				}

				var retErrVal reflect.Value
				if retErr == nil {
					retErrVal = reflect.Zero(reflect.TypeOf(new(error)).Elem())
				} else {
					retErrVal = reflect.ValueOf(retErr)
				}

				/*fmt.Println(req)*/
				//暂时不管resp 转 relflect.value
				return []reflect.Value{retVal, retErrVal}
			}

			//这里就是捕捉本地调用，而后调佣Set方法篡改了它， 改成发起RPC调用
			fnVal := reflect.MakeFunc(fieldTyp.Type, fn)

			fieldVal.Set(fnVal)

		}

	}

	return nil
}

type Client struct {
	//addr string
	pool      pool.Pool
	serialize serialize.Serializer
}

type ClientOption func(client *Client)

func ClientWithSerializer(sl serialize.Serializer) ClientOption {
	return func(client *Client) {
		client.serialize = sl
	}
}

func NewClient(address string, opts ...ClientOption) (*Client, error) {
	p, err := pool.NewChannelPool(&pool.Config{
		MaxCap:     30,
		InitialCap: 5,
		MaxIdle:    10,
		Factory: func() (interface{}, error) {
			return net.DialTimeout("tcp", address, 3*time.Second)
		},
		Close: func(i interface{}) error {
			return i.(net.Conn).Close()
		},
		IdleTimeout: time.Minute,
	})

	if err != nil {
		return nil, err
	}
	res := &Client{
		pool:      p,
		serialize: &json2.Serializer{},
	}

	for _, opt := range opts {
		opt(res)
	}
	return res, nil
}

func (c *Client) doInvoke(ctx context.Context, req *message.Request) (*message.Response, error) {
	data := message.EncodeReq(req)

	resp, err := c.send(ctx, data)
	if err != nil {
		return nil, err
	}
	return message.DecodeResp(resp), nil
}

func (c *Client) Invoke(ctx context.Context, req *message.Request) (*message.Response, error) {

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	ch := make(chan struct{}, 1)
	var (
		resp *message.Response
		err  error
	)

	go func() {
		resp, err = c.doInvoke(ctx, req)
		ch <- struct{}{}
		close(ch)
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-ch:
		return resp, err
	}

}

const numOfLengthBytes = 8

func (c *Client) Send(data []byte) ([]byte, error) {
	//conn, err := net.DialTimeout("tcp", c.addr, time.Second*3)
	val, err := c.pool.Get()
	if err != nil {
		return nil, err
	}
	conn := val.(net.Conn)
	defer func() {
		c.pool.Put(val)
	}()

	_, err = conn.Write(data)
	if err != nil {
		return nil, err
	}

	return ReadMsg(conn)

}

func (c *Client) send(ctx context.Context, data []byte) ([]byte, error) {
	//conn, err := net.DialTimeout("tcp", c.addr, time.Second*3)
	val, err := c.pool.Get()
	if err != nil {
		return nil, err
	}
	conn := val.(net.Conn)
	defer func() {
		c.pool.Put(val)
	}()

	_, err = conn.Write(data)
	if err != nil {
		return nil, err
	}

	if IsOneWay(ctx) {
		return nil, errors.New("micro : 这是一个 oneway 调用， 你不应该处理结果")
	}

	return ReadMsg(conn)

}
