package net

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

func Connect(network, addr string) error {
	conn, err := net.DialTimeout(network, addr, time.Second*3)
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()
	i := 0
	for {
		i++
		if i > 10 {
			return nil
		}
		_, err := conn.Write([]byte("hello"))
		if err != nil {
			return err
		}

		res := make([]byte, 128)
		_, err = conn.Read(res)
		if err != nil {
			return err
		}
		fmt.Println(string(res))
	}
}

type Client struct {
	network string
	addr    string
}

func (c *Client) Send(data string) (string, error) {
	conn, err := net.DialTimeout(c.network, c.addr, time.Second*3)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = conn.Close()
	}()

	reqLen := len(data)
	req := make([]byte, reqLen+numOfLengthBytes)
	binary.BigEndian.PutUint64(req[:numOfLengthBytes], uint64(reqLen))
	copy(req[numOfLengthBytes:], data)
	_, err = conn.Write(req)
	if err != nil {
		return "", err
	}

	lenBs := make([]byte, numOfLengthBytes)
	_, err = conn.Read(lenBs)
	if err != nil {
		return "", err
	}

	length := binary.BigEndian.Uint64(lenBs)
	respBs := make([]byte, length)
	_, err = conn.Read(respBs)
	if err != nil {
		return "", err
	}

	return string(respBs), nil

}
