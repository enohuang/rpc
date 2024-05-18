package net

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

type Pool struct {
	// 空闲连接队列
	idlesConnes chan *idleConn
	// 请求队列
	reqQueue []connReq

	// 最大链接书
	maxCnt int

	//当前链接数 - 你已经建好的
	cnt int

	//最大空闲时间
	maxIdleTime time.Duration

	//初始化连接数量
	//initCnt int

	//创建连接的方法
	factory func() (net.Conn, error)

	lock sync.Mutex
}

func NewPool(initCnt int, maxIdleCnt int, maxCnt int,
	maxIdleTime time.Duration,
	factory func() (net.Conn, error)) (*Pool, error) {

	if initCnt > maxIdleCnt {
		return nil, errors.New("micro: 初始连接数量不能大于最大空闲数")
	}

	idlesConns := make(chan *idleConn, maxIdleCnt)
	for i := 0; i < initCnt; i++ {
		conn, err := factory()
		if err != nil {
			return nil, err
		}
		idlesConns <- &idleConn{c: conn, lastActiveTime: time.Now()}
	}

	res := &Pool{
		idlesConnes: make(chan *idleConn, maxIdleCnt),
		maxCnt:      maxCnt,
		cnt:         0,
		//initCnt:     initCnt,
		factory: factory,
	}
	return res, nil
}

// Get ctx 控制超时
func (p *Pool) Get(ctx context.Context) (net.Conn, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	for {
		select {
		case ic := <-p.idlesConnes: //代表拿到空闲连接
			if ic.lastActiveTime.Add(p.maxIdleTime).Before(time.Now()) {
				_ = ic.c.Close()
				continue
			}
			return ic.c, nil
		default:
			//没有空闲连接
			p.lock.Lock()
			if p.cnt >= p.maxCnt {
				//超过上线，阻塞
				req := connReq{connChan: make(chan net.Conn, 1)}
				p.reqQueue = append(p.reqQueue, req)
				p.lock.Unlock()
				select {
				case <-ctx.Done():
					go func() {
						//超时，在这里转发
						c := <-req.connChan
						_ = p.Put(context.Background(), c)
					}()
					//选项1 ，从队列里面删除req 自己
					return nil, ctx.Err()
				//等别人归还
				case c := <-req.connChan:
					//可以用ping
					return c, nil
				}
			}
			p.lock.Unlock()
			c, err := p.factory()
			if err != nil {
				return nil, err
			}
			p.cnt++
			return c, nil
		}
	}

}

func (p *Pool) Put(ctx context.Context, cc net.Conn) error {
	p.lock.Lock()
	if len(p.reqQueue) > 0 {
		//有阻塞的请求
		req := p.reqQueue[0]
		p.reqQueue = p.reqQueue[1:]
		p.lock.Unlock()
		req.connChan <- cc
		return nil
	}
	defer p.lock.Unlock()
	ic := &idleConn{c: cc, lastActiveTime: time.Now()}
	select {
	case p.idlesConnes <- ic:
	default:
		_ = cc.Close()
		p.cnt--
	}
	return nil
}

type idleConn struct {
	c              net.Conn
	lastActiveTime time.Time
}

type connReq struct {
	connChan chan net.Conn
}
