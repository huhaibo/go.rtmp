// The MIT License (MIT)
//
// Copyright (c) 2014 winlin
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package rtmp

import (
	"bytes"
	"encoding/binary"
)

type RtmpBytesCodec interface {
	Next(n int) ([]byte)
	Read(p []byte) (v []byte)
	ReadByte() (v byte)
	ReadUInt16() (v uint16)
	ReadUInt24() (v uint32)
	ReadUInt32() (v uint32)
	ReadUInt32Le() (v uint32)
	TopUInt32() (v uint32)
	ReadString(n int) (string)
}
type RtmpStream interface {
	RtmpBytesCodec
	Reset(n int)
	Requires(n int) (bool)
}
type RtmpBuffer interface {
	RtmpBytesCodec
	EnsureBufferBytes(n int) (err error)
}
func NewRtmpBuffer(conn RtmpSocket) (RtmpBuffer) {
	r := &rtmpSocketBuffer{}
	r.conn = conn
	r.buffer = &bytes.Buffer{}
	return r
}
func NewRtmpStream(b []byte) (RtmpStream) {
	r := &rtmpSocketStream{}
	r.data = b
	r.buffer = bytes.NewBuffer(b)
	return r
}

// read data from socket if needed.
type rtmpBytesCodec struct{
	buffer *bytes.Buffer
}
type rtmpSocketBuffer struct{
	rtmpBytesCodec
	conn RtmpSocket
}
type rtmpSocketStream struct {
	rtmpBytesCodec
	data []byte
}

const RTMP_SOCKET_READ_SIZE = 4096

/**
* ensure the buffer contains n bytes, append from connection if needed.
 */
func (r *rtmpSocketBuffer) EnsureBufferBytes(n int) (err error) {
	var buffer *bytes.Buffer = r.buffer

	for buffer.Len() < n {
		buf := make([]byte, RTMP_SOCKET_READ_SIZE)

		var nsize int
		if nsize, err = r.conn.Read(buf); err != nil {
			return
		}

		if _, err = buffer.Write(buf[0:nsize]); err != nil {
			return
		}
	}

	return
}

// whether stream can satisfy the requires n bytes.
func (r *rtmpSocketStream) Requires(n int) (bool) {
	return r.buffer != nil && r.buffer.Len() >= n
}

// reset the decode buffer, start from index n
func (r *rtmpSocketStream) Reset(n int) {
	r.buffer = bytes.NewBuffer(r.data[n:])
}

// Next returns a slice containing the next n bytes from the buffer,
// advancing the buffer as if the bytes had been returned by Read.
// If there are fewer than n bytes in the buffer, Next returns the entire buffer.
// The slice is only valid until the next call to a read or write method.
func (r *rtmpBytesCodec) Next(n int) ([]byte){
	return r.buffer.Next(n)
}

// Read reads the next len(p) bytes from the buffer or until the buffer
// is drained.
func (r *rtmpBytesCodec) Read(p []byte) (v []byte) {
	if _, err := r.buffer.Read(p); err != nil {
		panic(err)
	}

	return p
}

// ReadByte reads and returns the next byte from the buffer.
func (r* rtmpBytesCodec) ReadByte() (v byte) {
	var err error

	if v, err = r.buffer.ReadByte(); err != nil {
		panic(err)
	}

	return v
}

// ReadByte reads and returns the next 3 bytes from the buffer. in big-endian
func (r* rtmpBytesCodec) ReadUInt24() (v uint32) {
	b := make([]byte, 4)
	if _, err := r.buffer.Read(b[1:]); err != nil {
		panic(err)
	}

	return binary.BigEndian.Uint32(b)
}

func (r* rtmpBytesCodec) ReadUInt16() (v uint16) {
	b := make([]byte, 2)
	if _, err := r.buffer.Read(b); err != nil {
		panic(err)
	}

	return binary.BigEndian.Uint16(b)
}

// ReadByte reads and returns the next 4 bytes from the buffer. in big-endian
func (r* rtmpBytesCodec) ReadUInt32() (v uint32) {
	b := make([]byte, 4)
	if _, err := r.buffer.Read(b); err != nil {
		panic(err)
	}

	return binary.BigEndian.Uint32(b)
}

// ReadByte reads and returns the next 4 bytes from the buffer. in little-endian
func (r* rtmpBytesCodec) ReadUInt32Le() (v uint32) {
	b := make([]byte, 4)
	if _, err := r.buffer.Read(b); err != nil {
		panic(err)
	}

	return binary.LittleEndian.Uint32(b)
}

// Get the first 4bytes, donot read it. in big-endian
func (r* rtmpBytesCodec) TopUInt32() (v uint32) {
	var b []byte = r.buffer.Bytes()
	b = b[0:4]

	return binary.BigEndian.Uint32(b)
}

// read string length specified by n.
func (r *rtmpBytesCodec) ReadString(n int) (v string) {
	b := make([]byte, n)
	if _, err := r.buffer.Read(b); err != nil {
		panic(err)
	}

	return string(b)
}
