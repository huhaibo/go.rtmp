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

/**
* the message header for RtmpMessage,
* the header can be used in chunk stream cache, for the chunk stream header.
* @see: RTMP 4.1. Message Header
*/
type RtmpMessageHeader struct {
}

/**
* the payload codec by the RtmpPacket.
* @see: RTMP 4.2. Message Payload
*/
type RtmpPacket struct {
}

/**
* the rtmp message, encode/decode to/from the rtmp stream,
* which contains a message header and a bytes payload.
* the header is RtmpMessageHeader, where the payload canbe decoded by RtmpPacket.
*/
type RtmpMessage struct {
}

/**
* incoming chunk stream maybe interlaced,
* use the chunk stream to cache the input RTMP chunk streams.
*/
type RtmpChunkStream struct {
	/**
	* represents the basic header fmt,
	* which used to identify the variant message header type.
	*/
	FMT byte
	/**
	* represents the basic header cid,
	* which is the chunk stream id.
	*/
	CId int
	/**
	* cached message header
	*/
	Header *RtmpMessageHeader
	/**
	* whether the chunk message header has extended timestamp.
	*/
	ExtendedTimestamp bool
	/**
	* partially read message.
	*/
	Msg *RtmpMessage
	/**
	* decoded msg count, to identify whether the chunk stream is fresh.
	*/
	MsgCount int64
}

/**
* the protocol provides the rtmp-message-protocol services,
* to recv RTMP message from RTMP chunk stream,
* and to send out RTMP message over RTMP chunk stream.
*/
type RtmpProtocol struct {
// peer in/out
	// the underlayer tcp connection, to read/write bytes from/to.
	conn *net.TCPConn
// peer in
	chunkStreams map[int]*RtmpChunkStream
}
