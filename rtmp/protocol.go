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
	"net"
)

func ReadBasicHeader(conn *net.TCPConn) (format byte, cid int, bh_size int, err error) {
	buf := make([]byte, 1)
	_, err = conn.Read(buf)
	if err != nil {
		return
	}

	format = buf[0]
	cid = int(format) & 0x3f
	format = (format >> 6) & 0x03
	bh_size = 1

	if cid == 0 {
		_, err = conn.Read(buf)
		if err != nil {
			return
		}
		cid = 64
		cid += int(buf[0])
		bh_size = 2
	} else if cid == 1 {
		buf = make([]byte, 2)
		_, err = conn.Read(buf)
		if err != nil {
			return
		}
		cid = 64
		cid += int(buf[0])
		cid += int(buf[1]) * 256
		bh_size = 3
	}

	return
}
