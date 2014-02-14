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
	"fmt"
	"net"
)

func (r *RtmpProtocol) Initialize(conn *net.TCPConn) (err error) {
	r.conn = conn
	r.chunkStreams = map[int]*RtmpChunkStream{}

	return
}

/**
* recv a message with raw/undecoded payload from peer.
* the payload is not decoded, use srs_rtmp_expect_message<T> if requires
* specifies message.
*/
func (r *RtmpProtocol) RecvMessage() (msg *RtmpMessage, err error) {
	format, cid, bh_size, err := r.read_basic_header()
	if err != nil {
		return
	}

	chunk, ok := r.chunkStreams[cid]
	if !ok {
		chunk = &RtmpChunkStream{CId:cid}
		r.chunkStreams[cid] = chunk
	}

	mh_size, err := r.read_message_header(chunk, format, bh_size)
	if err != nil {
		return
	}

	fmt.Println(mh_size)
	return
}

func (r *RtmpProtocol) read_message_header(chunk *RtmpChunkStream, format byte, bh_size int) (mh_size int, err error) {
	/**
	* we should not assert anything about fmt, for the first packet.
	* (when first packet, the chunk->msg is NULL).
	* the fmt maybe 0/1/2/3, the FMLE will send a 0xC4 for some audio packet.
	* the previous packet is:
	* 	04 			// fmt=0, cid=4
	* 	00 00 1a 	// timestamp=26
	*	00 00 9d 	// payload_length=157
	* 	08 			// message_type=8(audio)
	* 	01 00 00 00 // stream_id=1
	* the current packet maybe:
	* 	c4 			// fmt=3, cid=4
	* it's ok, for the packet is audio, and timestamp delta is 26.
	* the current packet must be parsed as:
	* 	fmt=0, cid=4
	* 	timestamp=26+26=52
	* 	payload_length=157
	* 	message_type=8(audio)
	* 	stream_id=1
	* so we must update the timestamp even fmt=3 for first packet.
	*/
	// fresh packet used to update the timestamp even fmt=3 for first packet.
	is_fresh_packet := false
	if chunk.Msg == nil {
		is_fresh_packet = true
	}

	// but, we can ensure that when a chunk stream is fresh,
	// the fmt must be 0, a new stream.
	if chunk.MsgCount == 0 && format != RTMP_FMT_TYPE0 {
		err = RtmpError{code:ERROR_RTMP_CHUNK_START, desc:"protocol error, fmt of first chunk must be 0"}
		return
	}

	// when exists cache msg, means got an partial message,
	// the fmt must not be type0 which means new message.
	if chunk.Msg != nil && format == RTMP_FMT_TYPE0 {
		err = RtmpError{code:ERROR_RTMP_CHUNK_START, desc:"protocol error, unexpect start of new chunk"}
		return
	}

	// create msg when new chunk stream start
	if chunk.Msg == nil {
		chunk.Msg = new(RtmpMessage)
	}

	if is_fresh_packet {
	}

	return
}

func (r *RtmpProtocol) read_basic_header() (format byte, cid int, bh_size int, err error) {
	buf := make([]byte, 1)
	_, err = r.conn.Read(buf)
	if err != nil {
		return
	}

	format = buf[0]
	cid = int(format) & 0x3f
	format = (format >> 6) & 0x03
	bh_size = 1

	if cid == 0 {
		_, err = r.conn.Read(buf)
		if err != nil {
			return
		}
		cid = 64
		cid += int(buf[0])
		bh_size = 2
	} else if cid == 1 {
		buf = make([]byte, 2)
		_, err = r.conn.Read(buf)
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
