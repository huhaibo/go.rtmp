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
	"math/rand"
	"time"
)

/**
* the rtmp message, encode/decode to/from the rtmp stream,
* which contains a message header and a bytes payload.
* the header is RtmpMessageHeader, where the payload canbe decoded by RtmpPacket.
*/
// @see: ISrsMessage, SrsCommonMessage, SrsSharedPtrMessage
type RtmpMessage struct {
	// 4.1. Message Header
	Header *RtmpMessageHeader
	// 4.2. Message Payload
	/**
	* The other part which is the payload is the actual data that is
	* contained in the message. For example, it could be some audio samples
	* or compressed video data. The payload format and interpretation are
	* beyond the scope of this document.
	*/
	Payload []byte
	/**
	* the payload is received from connection,
	* when len(Payload) == ReceivedPayloadLength, message receive completed.
	 */
	ReceivedPayloadLength int
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
func NewRtmpChunkStream(cid int) (r *RtmpChunkStream) {
	r = new(RtmpChunkStream)

	r.CId = cid
	r.Header = new(RtmpMessageHeader)

	return
}

/**
* the message header for RtmpMessage,
* the header can be used in chunk stream cache, for the chunk stream header.
* @see: RTMP 4.1. Message Header
*/
type RtmpMessageHeader struct {
	/**
	* One byte field to represent the message type. A range of type IDs
	* (1-7) are reserved for protocol control messages.
	*/
	MessageType byte
	/**
	* Three-byte field that represents the size of the payload in bytes.
	* It is set in big-endian format.
	*/
	PayloadLength uint32
	/**
	* Three-byte field that contains a timestamp delta of the message.
	* The 3 bytes are packed in the big-endian order.
	* @remark, only used for decoding message from chunk stream.
	*/
	TimestampDelta uint32
	/**
	* Four-byte field that identifies the stream of the message. These
	* bytes are set in little-endian format.
	*/
	StreamId uint32

	/**
	* Four-byte field that contains a timestamp of the message.
	* The 4 bytes are packed in the big-endian order.
	* @remark, used as calc timestamp when decode and encode time.
	* @remark, we use 64bits for large time for jitter detect and hls.
	*/
	Timestamp uint64
}

/**
* the handshake data, 6146B = 6KB,
* store in the protocol and never delete it for every connection.
 */
type RtmpHandshake struct {
	c0c1 []byte // 1537B
	s0s1s2 []byte // 3073B
	c2 []byte // 1536B
}

type RtmpAckWindowSize struct {
	ack_window_size uint32
	acked_size uint64
}

/**
* the protocol provides the rtmp-message-protocol services,
* to recv RTMP message from RTMP chunk stream,
* and to send out RTMP message over RTMP chunk stream.
*/
type RtmpProtocol struct {
// handshake
	handshake *RtmpHandshake
// peer in/out
	// the underlayer tcp connection, to read/write bytes from/to.
	conn *RtmpSocket
// peer in
	chunkStreams map[int]*RtmpChunkStream
	// the bytes read from underlayer tcp connection,
	// used for parse to RTMP message or packets.
	buffer *RtmpBuffer
	// input chunk stream chunk size.
	inChunkSize int32
	// the acked size
	inAckSize RtmpAckWindowSize
// peer out
	// output chunk stream chunk size.
	outChunkSize int32
}
/**
* create the rtmp protocol.
 */
func NewRtmpProtocol(conn *net.TCPConn) (r *RtmpProtocol, err error) {
	r = new(RtmpProtocol)

	r.conn = NewRtmpSocket(conn)
	r.chunkStreams = map[int]*RtmpChunkStream{}
	r.buffer = NewRtmpBuffer(r.conn)
	r.handshake = new(RtmpHandshake)

	r.inChunkSize = RTMP_DEFAULT_CHUNK_SIZE
	r.outChunkSize = r.inChunkSize

	rand.Seed(time.Now().UnixNano())

	return
}

/**
* the payload codec by the RtmpPacket.
* @see: RTMP 4.2. Message Payload
*/
/**
* the decoded message payload.
* @remark we seperate the packet from message,
*		for the packet focus on logic and domain data,
*		the message bind to the protocol and focus on protocol, such as header.
* 		we can merge the message and packet, using OOAD hierachy, packet extends from message,
* 		it's better for me to use components -- the message use the packet as payload.
*/
// @see: SrsPacket
type RtmpPacket interface {
	/**
	* decode functions.
	*/
	Decode([]byte) (error)
}

type RtmpConnectAppPacket struct {
}
