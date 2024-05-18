package net

import (
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"net"
	"os"
	"rpc/net/mocks"
	"testing"
	"time"
)

// mockgen -destination=micro/net/mocks/net_conn.gen.go -package=mocks net Conn
func TestHandleConn(t *testing.T) {
	testCases := []struct {
		name    string
		mock    func(ctrl *gomock.Controller) net.Conn
		wantErr error
	}{
		{
			name: "read error",
			mock: func(ctrl *gomock.Controller) net.Conn {
				conn := mocks.NewMockConn(ctrl)
				conn.EXPECT().Read(gomock.Any()).Return(0, errors.New("read error"))
				return conn
			},
			wantErr: errors.New("read error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			err := handleConn(tc.mock(ctrl))
			assert.Equal(t, tc.wantErr, err)

		})
	}
}

func TestClose(t *testing.T) {
	ch := make(chan struct{})
	go func() {
		time.Sleep(3 * time.Second)
		close(ch)
	}()
	file, _ := os.Create("name.txt")
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ch: //当一旦关闭ch， <- 永远返回
			file.WriteString("c\n")
		case <-ticker.C:
			file.WriteString("1\n")
		}
	}

}

func TestA(t *testing.T) {

	cs := make([]int, 4)
	l := len(cs)
	for i, _ := range cs {
		cs[i] = i
	}

	a := cs[0]
	copy(cs, cs[1:])
	cs = cs[:l-1]
	fmt.Println(a, cs)

}
