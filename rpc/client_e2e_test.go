package rpc

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"rpc/proto/gen"
	"rpc/rpc/serialize/json"
	"rpc/rpc/serialize/proto"
	"testing"
	"time"
)

func TestInitClientProxy(t *testing.T) {

	server := NewServer()

	service := &UserServiceServer{}

	server.RegisterService(service)
	go func() {
		err := server.Start("tcp", ":8081")
		t.Log(err)
	}()

	time.Sleep(3 * time.Second)
	usClient := &UserService{}

	client, er := NewClient(":8081")
	require.NoError(t, er)
	err := client.InitClientProxy(usClient)
	require.NoError(t, err)

	testCases := []struct {
		name string
		mock func()

		wantErr  error
		wantResp *GetIdResp
	}{
		{
			name: "no error",
			mock: func() {
				service.Err = nil
				service.Msg = "hello, world"
			},
			wantResp: &GetIdResp{
				Msg: "hello, world",
			},
		},

		{
			name: "both",
			mock: func() {
				service.Err = errors.New("mock error")
				service.Msg = "hello, world"
			},
			wantResp: &GetIdResp{
				Msg: "hello, world",
			},
			wantErr: errors.New("mock error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()

			//oneway 调用
			//resp, er := usClient.GetById(CtxWithOneWay(context.Background()), &GetIdReq{})
			// 我这边不能用resp

			//同步调用
			resp, er := usClient.GetById(context.Background(), &GetIdReq{Id: 123})

			/*异步调用  beg
			var respAsync *GetIdResp
			var wg sync.WaitGroup //开 waitgroup, chan控制
			wg.Add(1)
			go func() {
				respAsync, er = usClient.GetById(context.Background(), &GetIdReq{Id: 123})
				wg.Done()
			}()

			干了很多事
			 *************
			wg.Wait()
			_ = respAsync.Msg
			异步调用  end*/

			/*			回调 beg
						go func() {
							respAsync, err1 := usClient.GetById(context.Background(), &GetIdReq{Id: 123})
							//随便你怎么处理
							_ = err1
							_ = respAsync.Msg
						}()
						回调 end*/

			assert.Equal(t, tc.wantErr, er)
			assert.Equal(t, tc.wantResp, resp)
		})
	}

	/*resp, err := usClient.GetById(context.Background(), &GetIdReq{Id: 123})
	assert.Equal(t, &GetIdResp{Msg: "hello,world"}, resp)*/
}

func TestInitServiceProto(t *testing.T) {

	server := NewServer()

	service := &UserServiceServer{}

	server.RegisterService(service)
	server.RegisterSerializer(&json.Serializer{})
	server.RegisterSerializer(&proto.Serializer{})
	go func() {
		err := server.Start("tcp", ":8081")
		t.Log(err)
	}()

	time.Sleep(3 * time.Second)
	usClient := &UserService{}

	client, er := NewClient(":8081", ClientWithSerializer(&proto.Serializer{}))
	require.NoError(t, er)
	err := client.InitClientProxy(usClient)
	require.NoError(t, err)

	testCases := []struct {
		name string
		mock func()

		wantErr  error
		wantResp *GetIdResp
	}{
		{
			name: "no error",
			mock: func() {
				service.Err = nil
				service.Msg = "hello, world"
			},
			wantResp: &GetIdResp{
				Msg: "hello, world",
			},
		},

		{
			name: "both",
			mock: func() {
				service.Err = errors.New("mock error")
				service.Msg = "hello, world"
			},
			wantResp: &GetIdResp{
				Msg: "hello, world",
			},
			wantErr: errors.New("mock error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			resp, er := usClient.GetByIdProto(context.Background(), &gen.GetByIdReq{Id: 123})
			assert.Equal(t, tc.wantErr, er)
			if er != nil {
				return
			}
			if resp != nil && resp.User != nil {
				assert.Equal(t, tc.wantResp.Msg, resp.User.Name)
			}

		})
	}

	/*resp, err := usClient.GetById(context.Background(), &GetIdReq{Id: 123})
	assert.Equal(t, &GetIdResp{Msg: "hello,world"}, resp)*/
}

func TestInitClientWithOneWayProxy(t *testing.T) {

	server := NewServer()

	service := &UserServiceServer{}

	server.RegisterService(service)
	go func() {
		err := server.Start("tcp", ":8081")
		t.Log(err)
	}()

	time.Sleep(3 * time.Second)
	usClient := &UserService{}

	client, er := NewClient(":8081")
	require.NoError(t, er)
	err := client.InitClientProxy(usClient)
	require.NoError(t, err)

	testCases := []struct {
		name string
		mock func()

		wantErr  error
		wantResp *GetIdResp
	}{
		{
			name: "oneway",
			mock: func() {
				service.Err = errors.New("mock error")
				service.Msg = "hello, world"
			},
			wantResp: &GetIdResp{},
			wantErr:  errors.New("micro : 这是一个 oneway 调用， 你不应该处理结果"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			ctx := CtxWithOneWay(context.Background())
			resp, er := usClient.GetById(ctx, &GetIdReq{Id: 123})
			assert.Equal(t, tc.wantErr, er)
			assert.Equal(t, tc.wantResp, resp)

			//assert.Equal(t, tc.wantResp.Msg, resp.User.Name)
		})
	}

}

func TestTimeOut(t *testing.T) {

	server := NewServer()

	service := &UserServiceServerTimeOut{}

	server.RegisterService(service)
	go func() {
		err := server.Start("tcp", ":8081")
		t.Log(err)
	}()

	time.Sleep(3 * time.Second)
	usClient := &UserService{}

	client, er := NewClient(":8081")
	require.NoError(t, er)
	err := client.InitClientProxy(usClient)
	require.NoError(t, err)

	testCases := []struct {
		name string
		mock func() context.Context

		wantErr  error
		wantResp *GetIdResp
	}{
		{
			name: "timeout",
			mock: func() context.Context {
				service.Err = errors.New("mock error")
				service.Msg = "hello, world"
				//服务睡眠2秒 超时设置1秒
				service.sleep = time.Second * 2
				service.t = t
				ctx, _ := context.WithTimeout(context.Background(), time.Second)
				return ctx
			},
			wantResp: &GetIdResp{},
			wantErr:  context.DeadlineExceeded,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tc.mock()
			resp, er := usClient.GetById(ctx, &GetIdReq{Id: 123})
			assert.Equal(t, tc.wantErr, er)
			assert.Equal(t, tc.wantResp, resp)

			//assert.Equal(t, tc.wantResp.Msg, resp.User.Name)
		})
	}

}
