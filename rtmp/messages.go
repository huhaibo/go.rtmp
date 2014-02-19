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
* the header is MessageHeader, where the payload canbe decoded by RtmpPacket.
*/
// @see: ISrsMessage, SrsCommonMessage, SrsSharedPtrMessage
type Message struct {
	// 4.1. Message Header
	Header *MessageHeader
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
	/**
	* get the perfered cid(chunk stream id) which sendout over.
	* set at decoding, and canbe used for directly send message,
	* for example, dispatch to all connections.
	* @see: SrsSharedPtrMessage.SrsSharedPtr.perfer_cid
	*/
	PerferCid int
	/**
	* the payload sent length.
	 */
	SentPayloadLength int
}
func NewMessage() (*Message) {
	r := &Message{}
	r.Header = &MessageHeader{}
	return r
}

/**
* incoming chunk stream maybe interlaced,
* use the chunk stream to cache the input RTMP chunk streams.
*/
type ChunkStream struct {
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
	Header *MessageHeader
	/**
	* whether the chunk message header has extended timestamp.
	*/
	ExtendedTimestamp bool
	/**
	* partially read message.
	*/
	Msg *Message
	/**
	* decoded msg count, to identify whether the chunk stream is fresh.
	*/
	MsgCount int64
}
func NewChunkStream(cid int) (r *ChunkStream) {
	r = &ChunkStream{}

	r.CId = cid
	r.Header = &MessageHeader{}

	return
}

/**
* the message header for Message,
* the header can be used in chunk stream cache, for the chunk stream header.
* @see: RTMP 4.1. Message Header
*/
type MessageHeader struct {
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
type Handshake struct {
	c0c1 []byte // 1537B
	s0s1s2 []byte // 3073B
	c2 []byte // 1536B
}

type AckWindowSize struct {
	ack_window_size uint32
	acked_size uint64
}

/**
* max rtmp header size:
* 	1bytes basic header,
* 	11bytes message header,
* 	4bytes timestamp header,
* that is, 1+11+4=16bytes.
*/
const RTMP_MAX_FMT0_HEADER_SIZE = 16
/**
* max rtmp header size:
* 	1bytes basic header,
* 	4bytes timestamp header,
* that is, 1+4=5bytes.
*/
const RTMP_MAX_FMT3_HEADER_SIZE = 5

type Protocol interface {
	/**
	* do simple handshake with client, user can try simple/complex interlace,
	* that is, try complex handshake first, use simple if complex handshake failed.
	 */
	SimpleHandshake2Client() (err error)
	/**
	* recv message from connection.
	* the payload of message is []byte, user can decode it by DecodeMessage.
	 */
	//RecvMessage() (msg *Message, err error)
	/**
	* decode the received message to pkt.
	 */
	//DecodeMessage(msg *Message) (pkt interface {}, err error)
	/**
	* expect specified message by v, where v must be a ptr,
	* protocol stack will RecvMessage from connection and convert/set msg to v
	* if type matched, or drop the message and try again.
	 */
	ExpectMessage(v interface {}) (msg *Message, err error)
	/**
	* encode the packet to message, then send out by SendMessage.
	* return the cid which packet prefer.
	 */
	//EncodeMessage(pkt Encoder) (cid int, msg *Message, err error)
	/**
	* send message to peer over rtmp connection.
	* if pkt is Encoder, encode the pkt to Message and send out.
	* if pkt is Message already, directly send it out.
	 */
	SendPacket(pkt Encoder, stream_id uint32) (err error)
	SendMessage(pkt *Message, stream_id uint32) (err error)
}
/**
* create the rtmp protocol.
 */
func NewProtocol(conn *net.TCPConn) (Protocol, error) {
	r := &protocol{}

	r.conn = NewSocket(conn)
	r.chunkStreams = map[int]*ChunkStream{}
	r.buffer = NewRtmpBuffer(r.conn)
	r.handshake = &Handshake{}

	r.inChunkSize = RTMP_DEFAULT_CHUNK_SIZE
	r.outChunkSize = r.inChunkSize
	r.outHeaderFmt0 = make([]byte, RTMP_MAX_FMT0_HEADER_SIZE)
	r.outHeaderFmt3 = make([]byte, RTMP_MAX_FMT3_HEADER_SIZE)

	rand.Seed(time.Now().UnixNano())

	return r, nil
}

/**
* the protocol provides the rtmp-message-protocol services,
* to recv RTMP message from RTMP chunk stream,
* and to send out RTMP message over RTMP chunk stream.
*/
type protocol struct {
// handshake
	handshake *Handshake
// peer in/out
	// the underlayer tcp connection, to read/write bytes from/to.
	conn *Socket
// peer in
	chunkStreams map[int]*ChunkStream
	// the bytes read from underlayer tcp connection,
	// used for parse to RTMP message or packets.
	buffer *Buffer
	// input chunk stream chunk size.
	inChunkSize int32
	// the acked size
	inAckSize AckWindowSize
// peer out
	// output chunk stream chunk size.
	outChunkSize int32
	// bytes cache, size is RTMP_MAX_FMT0_HEADER_SIZE
	outHeaderFmt0 []byte
	// bytes cache, size is RTMP_MAX_FMT3_HEADER_SIZE
	outHeaderFmt3 []byte
}

/**
* the payload codec by the RtmpPacket.
* @see: RTMP 4.2. Message Payload
*/
// @see: SrsPacket
/**
* the decoded message payload.
* @remark we seperate the packet from message,
*		for the packet focus on logic and domain data,
*		the message bind to the protocol and focus on protocol, such as header.
* 		we can merge the message and packet, using OOAD hierachy, packet extends from message,
* 		it's better for me to use components -- the message use the packet as payload.
*/
type Decoder interface {
	/**
	* decode the packet from the s, which is created by rtmp message.
	 */
	Decode(s *Buffer) (err error)
}
/**
* encode the rtmp packet to payload of rtmp message.
 */
type Encoder interface {
	/**
	* get the rtmp chunk cid the packet perfered.
	 */
	GetPerferCid() (v int)
	/**
	* get packet message type
	 */
	GetMessageType() (v byte)
	/**
	* get the size of packet, to create the *HPBuffer.
	 */
	GetSize() (v int)
	/**
	* encode the packet to s, which is created by size=GetSize()
	 */
	Encode(s *Buffer) (err error)
}
func DecodePacket(r Protocol, header *MessageHeader, payload []byte) (packet interface {}, err error) {
	var pkt Decoder= nil
	var stream *Buffer = NewRtmpStream(payload)

	// decode specified packet type
	if header.IsAmf0Command() || header.IsAmf3Command() || header.IsAmf0Data() || header.IsAmf3Data() {
		// skip 1bytes to decode the amf3 command.
		if header.IsAmf3Command() &&  stream.Requires(1) {
			stream.Next(1)
		}

		amf0_codec := NewAmf0Codec(stream)

		// amf0 command message.
		// need to read the command name.
		var command string
		if command, err = amf0_codec.ReadString(); err != nil {
			return
		}

		// result/error packet
		if command == AMF0_COMMAND_RESULT || command == AMF0_COMMAND_ERROR {
			// TODO: FIXME: implements it
		}

		// reset to zero(amf3 to 1) to restart decode.
		if header.IsAmf3Command() &&  stream.Requires(1) {
			stream.Reset(1)
		} else {
			stream.Reset(0)
		}

		// decode command object.
		if command == AMF0_COMMAND_CONNECT {
			pkt = NewConnectAppPacket()
		}
		// TODO: FIXME: implements it
	} else if header.IsWindowAcknowledgementSize() {
		pkt =NewSetWindowAckSizePacket()
	}
	// TODO: FIXME: implements it

	if err == nil && pkt != nil {
		packet, err = pkt, pkt.Decode(stream)
	}

	return
}

/**
* 4.1.1. connect
* The client sends the connect command to the server to request
* connection to a server application instance.
*/
// @see: SrsConnectAppPacket
type ConnectAppPacket struct {
	CommandName string
	TransactionId float64
	CommandObject *Amf0Object
}
func NewConnectAppPacket() (*ConnectAppPacket) {
	return &ConnectAppPacket{ TransactionId:float64(1.0) }
}
// Decoder
func (r *ConnectAppPacket) Decode(s *Buffer) (err error) {
	codec := NewAmf0Codec(s)

	if r.CommandName, err = codec.ReadString(); err != nil {
		return
	}
	if r.CommandName != AMF0_COMMAND_CONNECT {
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 decode connect command_name failed."}
		return
	}

	if r.TransactionId, err = codec.ReadNumber(); err != nil {
		return
	}
	if r.TransactionId != 1.0 {
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 decode connect transaction_id failed."}
		return
	}

	if r.CommandObject, err = codec.ReadObject(); err != nil {
		return
	}
	if r.CommandObject == nil {
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 decode connect command_object failed."}
		return
	}

	return
}

/**
* response for SrsConnectAppPacket.
*/
// @see: SrsConnectAppResPacket
type ConnectAppResPacket struct {
	CommandName string
	TransactionId float64
	Props *Amf0Object
	Info *Amf0Object
}
func NewConnectAppResPacket() (*ConnectAppResPacket) {
	r := &ConnectAppResPacket{}
	r.CommandName = AMF0_COMMAND_RESULT
	r.TransactionId = float64(1.0)
	r.Props = NewAmf0Object()
	r.Info = NewAmf0Object()
	return r
}
func (r *ConnectAppResPacket) PropsSet(k string, v interface {}) (*ConnectAppResPacket) {
	// if empty or empty object, any value must has content.
	if a := ToAmf0(v); a != nil && a.Size() > 0 {
		r.Props.Set(k, a)
	}
	return r
}
func (r *ConnectAppResPacket) InfoSet(k string, v interface {}) (*ConnectAppResPacket) {
	// if empty or empty object, any value must has content.
	if a := ToAmf0(v); a != nil && a.Size() > 0 {
		r.Info.Set(k, a)
	}
	return r
}
// Encoder
func (r *ConnectAppResPacket) GetPerferCid() (v int) {
	return RTMP_CID_OverConnection
}
func (r *ConnectAppResPacket) GetMessageType() (v byte) {
	return RTMP_MSG_AMF0CommandMessage
}
func (r *ConnectAppResPacket) GetSize() (v int) {
	v = Amf0SizeString(r.CommandName)
	v += Amf0SizeNumber()
	v += r.Props.Size()
	v += r.Info.Size()
	return
}
func (r *ConnectAppResPacket) Encode(s *Buffer) (err error) {
	codec := NewAmf0Codec(s)

	if err = codec.WriteString(r.CommandName); err != nil {
		return
	}
	if err = codec.WriteNumber(r.TransactionId); err != nil {
		return
	}
	if r.Props.Size() > 0 {
		if err = codec.WriteObject(r.Props); err != nil {
			return
		}
	}
	if r.Info.Size() > 0 {
		if err = codec.WriteObject(r.Info); err != nil {
			return
		}
	}
	return
}

/**
* 5.5. Window Acknowledgement Size (5)
* The client or the server sends this message to inform the peer which
* window size to use when sending acknowledgment.
*/
// @see: SrsSetWindowAckSizePacket
type SetWindowAckSizePacket struct {
	AcknowledgementWindowSize uint32
}
func NewSetWindowAckSizePacket() (*SetWindowAckSizePacket) {
	return &SetWindowAckSizePacket{}
}
// Decoder
func (r *SetWindowAckSizePacket) Decode(s *Buffer) (err error) {
	if !s.Requires(4) {
		err = RtmpError{code:ERROR_RTMP_MESSAGE_DECODE, desc:"decode ack window size failed."}
		return
	}
	r.AcknowledgementWindowSize = s.ReadUInt32()
	return
}
// Encoder
func (r *SetWindowAckSizePacket) GetPerferCid() (v int) {
	return RTMP_CID_ProtocolControl
}
func (r *SetWindowAckSizePacket) GetMessageType() (v byte) {
	return RTMP_MSG_WindowAcknowledgementSize
}
func (r *SetWindowAckSizePacket) GetSize() (v int) {
	return 4
}
func (r *SetWindowAckSizePacket) Encode(s *Buffer) (err error) {
	if !s.Requires(4) {
		return RtmpError{code:ERROR_RTMP_MESSAGE_ENCODE, desc:"encode ack size packet failed."}
	}
	s.WriteUInt32(r.AcknowledgementWindowSize)
	return
}

/**
* 5.6. Set Peer Bandwidth (6)
* The client or the server sends this message to update the output
* bandwidth of the peer.
*/
// @see: SrsSetPeerBandwidthPacket
type SetPeerBandwidthPacket struct {
	Bandwidth uint32
	BandwidthType byte
}
// Encoder
func (r *SetPeerBandwidthPacket) GetPerferCid() (v int) {
	return RTMP_CID_ProtocolControl
}
func (r *SetPeerBandwidthPacket) GetMessageType() (v byte) {
	return RTMP_MSG_SetPeerBandwidth
}
func (r *SetPeerBandwidthPacket) GetSize() (v int) {
	return 5
}
func (r *SetPeerBandwidthPacket) Encode(s *Buffer) (err error) {
	if !s.Requires(5) {
		return RtmpError{code:ERROR_RTMP_MESSAGE_ENCODE, desc:"encode set bandwidth packet failed."}
	}
	s.WriteUInt32(r.Bandwidth).WriteByte(r.BandwidthType)
	return
}

/**
* 5.6. Set Peer Bandwidth (6)
* The client or the server sends this message to update the output
* bandwidth of the peer.
*/
// @see: SrsOnBWDonePacket
type OnBWDonePacket struct {
	CommandName string
	TransactionId float64
	Args *Amf0Any
}
func NewOnBWDonePacket() (*OnBWDonePacket) {
	r := &OnBWDonePacket{}
	r.CommandName = AMF0_COMMAND_ON_BW_DONE
	r.Args = ToAmf0Null()
	return r
}
// Encoder
func (r *OnBWDonePacket) GetPerferCid() (v int) {
	return RTMP_CID_OverConnection
}
func (r *OnBWDonePacket) GetMessageType() (v byte) {
	return RTMP_MSG_AMF0CommandMessage
}
func (r *OnBWDonePacket) GetSize() (v int) {
	return Amf0SizeString(r.CommandName) + Amf0SizeNumber() + Amf0SizeNullOrUndefined()
}
func (r *OnBWDonePacket) Encode(s *Buffer) (err error) {
	codec := NewAmf0Codec(s)
	if err = codec.WriteString(r.CommandName); err != nil {
		return
	}
	if err = codec.WriteNumber(r.TransactionId); err != nil {
		return
	}
	if err = codec.WriteNull(); err != nil {
		return
	}
	return
}
