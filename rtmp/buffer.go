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
	"math"
)

// read data from socket if needed.
type Buffer struct{
	// high performance buffer, to read/write from zero.
	buffer *HPBuffer
	// to read bytes and append to buffer.
	conn *Socket
}
func NewRtmpBuffer(conn *Socket) (*Buffer) {
	r := &Buffer{}
	r.conn = conn
	r.buffer = &HPBuffer{}
	return r
}
func NewRtmpStream(b []byte) (*Buffer) {
	r := &Buffer{}
	r.buffer = NewHPBuffer(b)
	return r
}

const RTMP_SOCKET_READ_SIZE = 4096

/**
* ensure the buffer contains n bytes, append from connection if needed.
 */
func (r *Buffer) EnsureBufferBytes(n int) (err error) {
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

func (r *Buffer) Consume(n int) (err error) {
	return r.buffer.Consume(n)
}

// whether stream can satisfy the requires n bytes.
func (r *Buffer) Requires(n int) (bool) {
	return r.buffer != nil && r.buffer.Len() >= n
}

// whether stream is empty
func (r *Buffer) Empty() (bool) {
	return r.buffer == nil || r.buffer.Len() <= 0
}

// reset the decode buffer, start from index n
func (r *Buffer) Reset() {
	r.buffer.Reset()
}

func (r *Buffer) Left() (int) {
	return r.buffer.Len()
}

// Next returns a slice containing the next n bytes from the buffer,
// advancing the buffer as if the bytes had been returned by Read.
// If there are fewer than n bytes in the buffer, Next returns the entire buffer.
// The slice is only valid until the next call to a read or write method.
func (r *Buffer) Skip(n int){
	if err := r.buffer.Skip(n); err != nil {
		panic(err)
	}
	return
}

// Read reads the next len(p) bytes from the buffer or until the buffer
// is drained.
func (r *Buffer) Read(n int) (b []byte) {
	b = r.buffer.Bytes()
	b = b[0:n]

	if err := r.buffer.Skip(n); err != nil {
		panic(err)
	}
	return
}

// ReadByte reads and returns the next byte from the buffer.
func (r* Buffer) ReadByte() (v byte) {
	b := r.buffer.Bytes()
	v = b[0]

	if err := r.buffer.Skip(1); err != nil {
		panic(err)
	}
	return v
}

// ReadByte reads and returns the next 3 bytes from the buffer. in big-endian
func (r* Buffer) ReadUInt24() (v uint32) {
	b := r.buffer.Bytes()
	v = uint32(b[2]) | uint32(b[1])<<8 | uint32(b[0])<<16
	//v = v & 0x00FFFFFF

	if err := r.buffer.Skip(3); err != nil {
		panic(err)
	}
	return v
}

func (r* Buffer) ReadUInt16() (v uint16) {
	b := r.buffer.Bytes()
	v = uint16(b[1]) | uint16(b[0])<<8

	if err := r.buffer.Skip(2); err != nil {
		panic(err)
	}
	return v
}

// ReadByte reads and returns the next 4 bytes from the buffer. in big-endian
func (r* Buffer) ReadUInt32() (v uint32) {
	b := r.buffer.Bytes()
	v = uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24

	if err := r.buffer.Skip(4); err != nil {
		panic(err)
	}
	return v
}

// ReadByte reads and returns the next 8 bytes from the buffer. in big-endian
func (r* Buffer) ReadFloat64() (v float64) {
	b := r.buffer.Bytes()
	v64 := uint64(b[7]) | uint64(b[6])<<8 | uint64(b[5])<<16 | uint64(b[4])<<24 |
		uint64(b[3])<<32 | uint64(b[2])<<40 | uint64(b[1])<<48 | uint64(b[0])<<56
	v = math.Float64frombits(v64)

	if err := r.buffer.Skip(8); err != nil {
		panic(err)
	}
	return v
}

// ReadByte reads and returns the next 4 bytes from the buffer. in little-endian
func (r* Buffer) ReadUInt32Le() (v uint32) {
	b := r.buffer.Bytes()
	v = uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24

	if err := r.buffer.Skip(4); err != nil {
		panic(err)
	}
	return v
}

func (r *Buffer) Write(v []byte) (*Buffer) {
	if _, err := r.buffer.Write(v); err != nil {
		panic(err)
	}

	return r
}

func (r *Buffer) WriteByte(v byte) (*Buffer) {
	b := r.buffer.Bytes()
	b[0] = v

	if err := r.buffer.Skip(1); err != nil {
		panic(err)
	}
	return r
}

// ReadByte reads and returns the next 4 bytes from the buffer. in big-endian
func (r *Buffer) WriteUInt32(v uint32) (*Buffer) {
	b := r.buffer.Bytes()
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)

	if err := r.buffer.Skip(4); err != nil {
		panic(err)
	}
	return r
}

func (r *Buffer) WriteUInt24(v uint32) (*Buffer) {
	b := r.buffer.Bytes()
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)

	if err := r.buffer.Skip(3); err != nil {
		panic(err)
	}
	return r
}

func (r *Buffer) WriteUInt16(v uint16) (*Buffer) {
	b := r.buffer.Bytes()
	b[0] = byte(v >> 8)
	b[1] = byte(v)

	if err := r.buffer.Skip(2); err != nil {
		panic(err)
	}
	return r
}

func (r *Buffer) WriteUInt32Le(v uint32) (*Buffer) {
	b := r.buffer.Bytes()
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)

	if err := r.buffer.Skip(4); err != nil {
		panic(err)
	}
	return r
}

func (r *Buffer) WriteFloat64(v64 float64) (*Buffer) {
	v := math.Float64bits(v64)

	b := r.buffer.Bytes()
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)

	if err := r.buffer.Skip(8); err != nil {
		panic(err)
	}
	return r
}
