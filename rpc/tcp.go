package rpc

import (
	"encoding/binary"
	"net"
)

func ReadMsg(conn net.Conn) ([]byte, error) {
	//头部的头和协议体
	lenBs := make([]byte, numOfLengthBytes)
	_, err := conn.Read(lenBs)
	if err != nil {
		return nil, err
	}

	headerLength := binary.BigEndian.Uint32(lenBs[:4])
	bodyLength := binary.BigEndian.Uint32(lenBs[4:8])
	length := headerLength + bodyLength
	data := make([]byte, length)
	_, err = conn.Read(data[8:])
	copy(data[:8], lenBs)

	return data, nil
}

/*func EncodeMsg(data []byte) ([]byte, error) {
	reqLen := len(data)
	res := make([]byte, reqLen+numOfLengthBytes)
	binary.BigEndian.PutUint64(res[:numOfLengthBytes], uint64(reqLen))
	copy(res[numOfLengthBytes:], data)
	return res, nil
}*/
