package rpc

import (
	"context"
	"log"
	"rpc/proto/gen"
	"testing"
	"time"
)

type UserService struct {
	//用发射来赋值
	//类型是函数式的字段, 它不是方法
	GetById func(ctx context.Context, req *GetIdReq) (*GetIdResp, error)

	GetByIdProto func(ctx context.Context, req *gen.GetByIdReq) (*gen.GetByIdResp, error)
}

func (u UserService) Name() string {
	return "user-service"
}

type GetIdReq struct {
	Id int
}

type GetIdResp struct {
	Msg string
}

type UserServiceServer struct {
	Err error
	Msg string
}

func (u *UserServiceServer) Name() string {
	return "user-service"
}

func (u *UserServiceServer) GetById(ctx context.Context, req *GetIdReq) (*GetIdResp, error) {
	log.Println(req)
	defer func() {
		log.Println("GetById return", u.Msg, u.Err)

	}()
	return &GetIdResp{Msg: u.Msg}, u.Err
}

func (u *UserServiceServer) GetByIdProto(ctx context.Context, req *gen.GetByIdReq) (*gen.GetByIdResp, error) {
	log.Println(req)
	defer func() {
		log.Println("GetByIdProto return", u.Msg, u.Err)

	}()
	return &gen.GetByIdResp{
		User: &gen.User{Name: u.Msg},
	}, u.Err
}

type UserServiceServerTimeOut struct {
	t     *testing.T
	sleep time.Duration
	Err   error
	Msg   string
}

func (u *UserServiceServerTimeOut) Name() string {
	return "user-service"
}

func (u *UserServiceServerTimeOut) GetById(ctx context.Context, req *GetIdReq) (*GetIdResp, error) {
	if _, ok := ctx.Deadline(); !ok {
		u.t.Fatal("没有设置超时")
	}
	time.Sleep(u.sleep)
	return &GetIdResp{Msg: u.Msg}, u.Err
}
