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
	"reflect"
)

// should ack the read, ack to peer
func (r *RtmpAckWindowSize) ShouldAckRead(n uint64) (bool) {
	if r.ack_window_size <= 0 {
		return false
	}

	return n - uint64(r.acked_size) > uint64(r.ack_window_size)
}

/**
* recv a message with raw/undecoded payload from peer.
* the payload is not decoded, use srs_rtmp_expect_message<T> if requires
* specifies message.
*/
func (r *rtmpProtocol) RecvMessage() (msg *RtmpMessage, err error) {
	for {
		if msg, err = r.recv_interlaced_message(); err != nil {
			return
		}

		if msg == nil {
			continue
		}

		if msg.ReceivedPayloadLength <= 0 || msg.Header.PayloadLength <= 0 {
			continue
		}

		if err = r.on_recv_message(msg); err != nil {
			return
		}

		break
	}
	return
}

/**
* expect a specified message by v, drop others util got specified one.
* for example:
* 		var pkt *RtmpConnectAppPacket
*		_, err = r.protocol.ExpectMessage(&pkt)
* 		// use the decoded pkt contains the connect app info.
*/
func (r *rtmpProtocol) ExpectMessage(v interface {}) (msg *RtmpMessage, err error) {
	rv := reflect.ValueOf(v)
	rt := reflect.TypeOf(v)
	if rv.Kind() != reflect.Ptr {
		err = RtmpError{code:ERROR_GO_REFLECT_PTR_REQUIRES, desc:"param must be ptr for expect message"}
		return
	}
	if rv.IsNil() {
		err = RtmpError{code:ERROR_GO_REFLECT_NEVER_NIL, desc:"param should never be nil"}
		return
	}
	if !rv.Elem().CanSet() {
		err = RtmpError{code:ERROR_GO_REFLECT_CAN_SET, desc:"param should be settable"}
		return
	}

	for {
		if msg, err = r.RecvMessage(); err != nil {
			return
		}
		if msg == nil || msg.Payload == nil{
			continue
		}

		var pkt RtmpPacket
		if pkt, err = ParseRtmpPacket(r, msg.Header, msg.Payload); err != nil {
			return
		}
		if pkt == nil {
			continue
		}

		// check the convertible and convert to the value or ptr value.
		// for example, the v like the c++ code: Msg**v
		pkt_rt := reflect.TypeOf(pkt)
		if pkt_rt.ConvertibleTo(rt) {
			// directly match, the pkt is like c++: Msg**pkt
			// set the v by: *v = *pkt
			rv.Elem().Set(reflect.ValueOf(pkt).Elem())
			return
		}
		if pkt_rt.ConvertibleTo(rt.Elem()) {
			// ptr match, the pkt is like c++: Msg*pkt
			// set the v by: *v = pkt
			rv.Elem().Set(reflect.ValueOf(pkt))
			return
		}
	}

	return
}

func (r *rtmpProtocol) on_recv_message(msg *RtmpMessage) (err error) {
	// acknowledgement
	if r.inAckSize.ShouldAckRead(r.conn.RecvBytes()) {
		return r.response_acknowledgement_message()
	}

	// TODO: FIXME: implements it

	return
}

func (r *rtmpProtocol) recv_interlaced_message() (msg *RtmpMessage, err error) {
	// chunk stream basic header.
	format, cid, _, err := r.read_basic_header()
	if err != nil {
		return
	}

	// get the cached chunk stream.
	chunk, ok := r.chunkStreams[cid]
	if !ok {
		chunk = NewRtmpChunkStream(cid)
		r.chunkStreams[cid] = chunk
	}

	// chunk stream message header
	if _, err = r.read_message_header(chunk, format); err != nil {
		return
	}

	// read msg payload from chunk stream.
	if msg, err = r.read_message_payload(chunk); err != nil {
		return
	}

	return
}

func (r *rtmpProtocol) read_basic_header() (format byte, cid int, bh_size int, err error) {
	if err = r.buffer.EnsureBufferBytes(1); err != nil {
		return
	}

	format = r.buffer.ReadByte()
	cid = int(format) & 0x3f
	format = (format >> 6) & 0x03
	bh_size = 1

	if cid == 0 {
		if err = r.buffer.EnsureBufferBytes(1); err != nil {
			return
		}
		cid = 64
		cid += int(r.buffer.ReadByte())
		bh_size = 2
	} else if cid == 1 {
		if err = r.buffer.EnsureBufferBytes(2); err != nil {
			return
		}

		cid = 64
		cid += int(r.buffer.ReadByte())
		cid += int(r.buffer.ReadByte()) * 256
		bh_size = 3
	}

	return
}

func (r *rtmpProtocol) read_message_header(chunk *RtmpChunkStream, format byte) (mh_size int, err error) {
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
		chunk.Msg = &RtmpMessage{}
	}

	// read message header from socket to buffer.
	mh_sizes := []int{11, 7, 3, 0}
	mh_size = mh_sizes[int(format)];
	if err = r.buffer.EnsureBufferBytes(mh_size); err != nil {
		return
	}

	// parse the message header.
	// see also: ngx_rtmp_recv
	if format <= RTMP_FMT_TYPE2 {
		chunk.Header.TimestampDelta = r.buffer.ReadUInt24()

		// fmt: 0
		// timestamp: 3 bytes
		// If the timestamp is greater than or equal to 16777215
		// (hexadecimal 0x00ffffff), this value MUST be 16777215, and the
		// ‘extended timestamp header’ MUST be present. Otherwise, this value
		// SHOULD be the entire timestamp.
		//
		// fmt: 1 or 2
		// timestamp delta: 3 bytes
		// If the delta is greater than or equal to 16777215 (hexadecimal
		// 0x00ffffff), this value MUST be 16777215, and the ‘extended
		// timestamp header’ MUST be present. Otherwise, this value SHOULD be
		// the entire delta.
		if chunk.ExtendedTimestamp = false; chunk.Header.TimestampDelta >= RTMP_EXTENDED_TIMESTAMP {
			chunk.ExtendedTimestamp = true
		}
		if chunk.ExtendedTimestamp {
			// Extended timestamp: 0 or 4 bytes
			// This field MUST be sent when the normal timsestamp is set to
			// 0xffffff, it MUST NOT be sent if the normal timestamp is set to
			// anything else. So for values less than 0xffffff the normal
			// timestamp field SHOULD be used in which case the extended timestamp
			// MUST NOT be present. For values greater than or equal to 0xffffff
			// the normal timestamp field MUST NOT be used and MUST be set to
			// 0xffffff and the extended timestamp MUST be sent.
			//
			// if extended timestamp, the timestamp must >= RTMP_EXTENDED_TIMESTAMP
			// we set the timestamp to RTMP_EXTENDED_TIMESTAMP to identify we
			// got an extended timestamp.
			chunk.Header.Timestamp = RTMP_EXTENDED_TIMESTAMP
		} else {
			if format == RTMP_FMT_TYPE0 {
				// 6.1.2.1. Type 0
				// For a type-0 chunk, the absolute timestamp of the message is sent
				// here.
				chunk.Header.Timestamp = uint64(chunk.Header.TimestampDelta)
			} else {
				// 6.1.2.2. Type 1
				// 6.1.2.3. Type 2
				// For a type-1 or type-2 chunk, the difference between the previous
				// chunk's timestamp and the current chunk's timestamp is sent here.
				chunk.Header.Timestamp += uint64(chunk.Header.TimestampDelta)
			}
		}

		if format <= RTMP_FMT_TYPE1 {
			chunk.Header.PayloadLength = r.buffer.ReadUInt24()

			// if msg exists in cache, the size must not changed.
			if chunk.Msg.Payload != nil && len(chunk.Msg.Payload) != int(chunk.Header.PayloadLength) {
				err = RtmpError{code:ERROR_RTMP_PACKET_SIZE, desc:"cached message size should never change"}
				return
			}

			chunk.Header.MessageType = r.buffer.ReadByte()

			if format == RTMP_FMT_TYPE0 {
				chunk.Header.StreamId = r.buffer.ReadUInt32Le()
			}
		}
	} else {
		// update the timestamp even fmt=3 for first stream
		if is_fresh_packet && !chunk.ExtendedTimestamp {
			chunk.Header.Timestamp += uint64(chunk.Header.TimestampDelta)
		}
	}

	if chunk.ExtendedTimestamp {
		mh_size += 4
		if err = r.buffer.EnsureBufferBytes(4); err != nil {
			return
		}

		// ffmpeg/librtmp may donot send this filed, need to detect the value.
		// @see also: http://blog.csdn.net/win_lin/article/details/13363699
		timestamp := r.buffer.TopUInt32()

		// compare to the chunk timestamp, which is set by chunk message header
		// type 0,1 or 2.
		if chunk.Header.Timestamp > RTMP_EXTENDED_TIMESTAMP && chunk.Header.Timestamp != uint64(timestamp) {
			mh_size -= 4
		} else {
			chunk.Header.Timestamp = uint64(timestamp)
			// consume the 4bytes timestamp.
			r.buffer.Next(4)
		}
	}

	// valid message
	if int32(chunk.Header.PayloadLength) < 0 {
		err = RtmpError{code:ERROR_RTMP_MSG_INVLIAD_SIZE, desc:"chunk packet should never be negative"}
		return
	}

	// copy header to msg
	copy := *chunk.Header
	chunk.Msg.Header = &copy

	// increase the msg count, the chunk stream can accept fmt=1/2/3 message now.
	chunk.MsgCount++

	return
}

func (r *rtmpProtocol) read_message_payload(chunk *RtmpChunkStream) (msg *RtmpMessage, err error) {
	// empty message
	if int32(chunk.Header.PayloadLength) <= 0 {
		msg = chunk.Msg
		chunk.Msg = nil
		return
	}

	// the chunk payload size.
	payload_size := int(chunk.Header.PayloadLength) - chunk.Msg.ReceivedPayloadLength
	payload_size = int(math.Min(float64(payload_size), float64(r.inChunkSize)))

	// create msg payload if not initialized
	if chunk.Msg.Payload == nil {
		chunk.Msg.Payload = make([]byte, chunk.Msg.Header.PayloadLength)
	}

	// read payload to buffer
	if err = r.buffer.EnsureBufferBytes(payload_size); err != nil {
		return
	}
	r.buffer.Read(chunk.Msg.Payload[chunk.Msg.ReceivedPayloadLength:chunk.Msg.ReceivedPayloadLength+payload_size])
	chunk.Msg.ReceivedPayloadLength += payload_size

	// got entire RTMP message?
	if chunk.Msg.ReceivedPayloadLength == len(chunk.Msg.Payload) {
		msg = chunk.Msg
		chunk.Msg = nil
		return
	}

	return
}

func (r *rtmpProtocol) response_acknowledgement_message() (err error) {
	// TODO: FIXME: implements it
	return
}

func (r *RtmpMessageHeader) IsAmf0Command() (bool) {
	return r.MessageType == RTMP_MSG_AMF0CommandMessage
}
