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

type RtmpStream interface {
	Next(n int) ([]byte)
	Read(p []byte) (v []byte)
	ReadByte() (v byte)
	ReadUInt24() (v uint32)
	ReadUInt32() (v uint32)
	ReadUInt32Le() (v uint32)
	TopUInt32() (v uint32)
}
type RtmpBuffer interface {
	RtmpStream
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
	r.buffer = bytes.NewBuffer(b)
	return r
}

// read data from socket if needed.
type rtmpSocketStream struct{
	buffer *bytes.Buffer
}
type rtmpSocketBuffer struct{
	rtmpSocketStream
	conn RtmpSocket
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

// Next returns a slice containing the next n bytes from the buffer,
// advancing the buffer as if the bytes had been returned by Read.
// If there are fewer than n bytes in the buffer, Next returns the entire buffer.
// The slice is only valid until the next call to a read or write method.
func (r *rtmpSocketStream) Next(n int) ([]byte){
	return r.buffer.Next(n)
}

// Read reads the next len(p) bytes from the buffer or until the buffer
// is drained.
func (r *rtmpSocketStream) Read(p []byte) (v []byte) {
	if _, err := r.buffer.Read(p); err != nil {
		panic(err)
	}

	return p
}

// ReadByte reads and returns the next byte from the buffer.
func (r* rtmpSocketStream) ReadByte() (v byte) {
	var err error

	if v, err = r.buffer.ReadByte(); err != nil {
		panic(err)
	}

	return v
}

// ReadByte reads and returns the next 3 bytes from the buffer. in big-endian
func (r* rtmpSocketStream) ReadUInt24() (v uint32) {
	buf := make([]byte, 4)

	if _, err := r.buffer.Read(buf[1:]); err != nil {
		panic(err)
	}

	return binary.BigEndian.Uint32(buf)
}

// ReadByte reads and returns the next 4 bytes from the buffer. in big-endian
func (r* rtmpSocketStream) ReadUInt32() (v uint32) {
	buf := make([]byte, 4)

	if _, err := r.buffer.Read(buf); err != nil {
		panic(err)
	}

	return binary.BigEndian.Uint32(buf)
}

// ReadByte reads and returns the next 4 bytes from the buffer. in little-endian
func (r* rtmpSocketStream) ReadUInt32Le() (v uint32) {
	buf := make([]byte, 4)

	if _, err := r.buffer.Read(buf); err != nil {
		panic(err)
	}

	return binary.LittleEndian.Uint32(buf)
}

// Get the first 4bytes, donot read it. in big-endian
func (r* rtmpSocketStream) TopUInt32() (v uint32) {
	var buf []byte = r.buffer.Bytes()
	buf = buf[0:4]

	return binary.BigEndian.Uint32(buf)
}
