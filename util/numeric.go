package util

import (
	"bytes"
	"encoding/binary"
)

// 整型转换成字节
func IntToBytes(n int) []byte {
	x := int64(n)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

// 字节转换成整型
func BytesToInt(b []byte) int {
	bytesBuffer := bytes.NewBuffer(b)
	var x int64
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return int(x)
}

// 把两个uint32拼接成一个uint64
func CombineUint32(a, b uint32) uint64 {
	return (uint64(a) << 32) | uint64(b)
}

// 把一个uint64拆成两个uint32
func DisassembleUint64(c uint64) (a, b uint32) {
	a = uint32(c >> 32)
	b = uint32(c << 32 >> 32)
	return
}
