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
	"encoding/binary"
	"math"
)

type RtmpBytesCodec interface {
	// all read methods donot return error,
	// if error, panic instead.
	Next(n int) ([]byte)
	Read(p []byte) (v []byte)
	ReadByte() (v byte)
	ReadUInt16() (v uint16)
	ReadUInt24() (v uint32)
	ReadUInt32() (v uint32)
	ReadUInt32Le() (v uint32)
	ReadFloat64() (v float64)
	ReadString(n int) (string)
	// all write methods return the interface itself,
	// if error, panic instead.
	WriteUInt32(v uint32) (RtmpBytesCodec)
	WriteUInt24(v uint32) (RtmpBytesCodec)
	WriteUInt32Le(v uint32) (RtmpBytesCodec)
	WriteByte(v byte) (RtmpBytesCodec)
}
type RtmpStream interface {
	RtmpBytesCodec
	Reset(n int)
	Empty() (bool)
	Left() (int)
	Requires(n int) (bool)
}
type RtmpBuffer interface {
	RtmpBytesCodec
	Truncate() (err error)
	EnsureBufferBytes(n int) (err error)
}
func NewRtmpBuffer(conn RtmpSocket) (RtmpBuffer) {
	r := &rtmpBuffer{}
	r.conn = conn
	r.buffer = &HPBuffer{}
	return r
}
func NewRtmpStream(b []byte) (RtmpStream) {
	r := &rtmpBuffer{}
	r.buffer = NewHPBuffer(b)
	return r
}

// read data from socket if needed.
type rtmpBuffer struct{
	buffer *HPBuffer
	conn RtmpSocket
}

const RTMP_SOCKET_READ_SIZE = 4096

/**
* ensure the buffer contains n bytes, append from connection if needed.
 */
func (r *rtmpBuffer) EnsureBufferBytes(n int) (err error) {
	var buffer *HPBuffer = r.buffer

	buf := make([]byte, RTMP_SOCKET_READ_SIZE)
	for buffer.Len() < n {
		var nsize int
		if nsize, err = r.conn.Read(buf); err != nil {
			return
		}

		if _, err = buffer.Append(buf[0:nsize]); err != nil {
			return
		}
	}

	return
}

func (r *rtmpBuffer) Truncate() (err error) {
	return r.buffer.Truncate()
}

// whether stream can satisfy the requires n bytes.
func (r *rtmpBuffer) Requires(n int) (bool) {
	return r.buffer != nil && r.buffer.Len() >= n
}

// whether stream is empty
func (r *rtmpBuffer) Empty() (bool) {
	return r.buffer == nil || r.buffer.Len() <= 0
}

// reset the decode buffer, start from index n
func (r *rtmpBuffer) Reset(n int) {
	r.buffer.Reset(n)
}

func (r *rtmpBuffer) Left() (int) {
	return r.buffer.Len()
}

// Next returns a slice containing the next n bytes from the buffer,
// advancing the buffer as if the bytes had been returned by Read.
// If there are fewer than n bytes in the buffer, Next returns the entire buffer.
// The slice is only valid until the next call to a read or write method.
func (r *rtmpBuffer) Next(n int) ([]byte){
	return r.buffer.Next(n)
}

// Read reads the next len(p) bytes from the buffer or until the buffer
// is drained.
func (r *rtmpBuffer) Read(p []byte) (v []byte) {
	if _, err := r.buffer.Read(p); err != nil {
		panic(err)
	}

	return p
}

// ReadByte reads and returns the next byte from the buffer.
func (r* rtmpBuffer) ReadByte() (v byte) {
	var err error

	if v, err = r.buffer.ReadByte(); err != nil {
		panic(err)
	}

	return v
}

// ReadByte reads and returns the next 3 bytes from the buffer. in big-endian
func (r* rtmpBuffer) ReadUInt24() (v uint32) {
	b := make([]byte, 4)
	if _, err := r.buffer.Read(b[1:]); err != nil {
		panic(err)
	}

	return binary.BigEndian.Uint32(b)
}

func (r* rtmpBuffer) ReadUInt16() (v uint16) {
	b := make([]byte, 2)
	if _, err := r.buffer.Read(b); err != nil {
		panic(err)
	}

	return binary.BigEndian.Uint16(b)
}

// ReadByte reads and returns the next 4 bytes from the buffer. in big-endian
func (r* rtmpBuffer) ReadUInt32() (v uint32) {
	b := make([]byte, 4)
	if _, err := r.buffer.Read(b); err != nil {
		panic(err)
	}

	return binary.BigEndian.Uint32(b)
}

// ReadByte reads and returns the next 8 bytes from the buffer. in big-endian
func (r* rtmpBuffer) ReadFloat64() (v float64) {
	b := make([]byte, 8)
	if _, err := r.buffer.Read(b); err != nil {
		panic(err)
	}

	v64 := binary.BigEndian.Uint64(b)
	v = math.Float64frombits(v64)

	return
}

// ReadByte reads and returns the next 4 bytes from the buffer. in little-endian
func (r* rtmpBuffer) ReadUInt32Le() (v uint32) {
	b := make([]byte, 4)
	if _, err := r.buffer.Read(b); err != nil {
		panic(err)
	}

	return binary.LittleEndian.Uint32(b)
}

// read string length specified by n.
func (r *rtmpBuffer) ReadString(n int) (v string) {
	b := make([]byte, n)
	if _, err := r.buffer.Read(b); err != nil {
		panic(err)
	}

	return string(b)
}

// ReadByte reads and returns the next 4 bytes from the buffer. in big-endian
func (r *rtmpBuffer) WriteUInt32(v uint32) (RtmpBytesCodec) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)

	if _, err := r.buffer.Write(b); err != nil {
		panic(err)
	}

	return r
}

func (r *rtmpBuffer) WriteUInt24(v uint32) (RtmpBytesCodec) {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	if _, err := r.buffer.Write(b[1:]); err != nil {
		panic(err)
	}

	return r
}

func (r *rtmpBuffer) WriteUInt32Le(v uint32) (RtmpBytesCodec) {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	if _, err := r.buffer.Write(b); err != nil {
		panic(err)
	}

	return r
}

func (r *rtmpBuffer) WriteByte(v byte) (RtmpBytesCodec) {
	if err := r.buffer.WriteByte(v); err != nil {
		panic(err)
	}

	return r
}
