package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"rpc/rpc/message"
	"testing"

	json2 "rpc/rpc/serialize/json"
)

func Test_setFuncField(t *testing.T) {
	testCases := []struct {
		name string

		mock    func(ctrl *gomock.Controller) Proxy
		service Service
		wantErr error
	}{
		{
			name:    "nil",
			service: nil,
			mock: func(ctrl *gomock.Controller) Proxy {
				return NewMockProxy(ctrl)
			},
			wantErr: errors.New("rpc : 不支持 nil"),
		},

		{
			name:    "no pointer",
			service: UserService{},
			mock: func(ctrl *gomock.Controller) Proxy {
				return NewMockProxy(ctrl)
			},
			wantErr: errors.New("rpc: 只支持指向结构体的一级指针"),
		},

		{
			name:    "user service",
			service: &UserService{},
			mock: func(ctrl *gomock.Controller) Proxy {
				p := NewMockProxy(ctrl)
				p.EXPECT().Invoke(gomock.Any(), &message.Request{
					ServiceName: "user-service",
					MethodName:  "GetById",
					Data: func() []byte {
						a := &GetIdReq{123}
						b, _ := json.Marshal(a)
						return b
					}(),
				}).Return(&message.Response{}, nil)
				return p
			},
		},
	}

	s := &json2.Serializer{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			err := setFuncField(tc.service, tc.mock(ctrl), s)
			if err != nil {
				return
			}
			resp, err := tc.service.(*UserService).GetById(context.Background(), &GetIdReq{Id: 123})
			assert.Equal(t, tc.wantErr, err)
			t.Log(resp)
		})
	}

}
