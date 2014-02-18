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
	"net/url"
	"strings"
	"strconv"
	"fmt"
)

const (
	RtmpCodecAMF0 = 0
	RtmpCodecAMF3 = 3
	RtmpDefaultPort = 1935
	// 5.6. Set Peer Bandwidth (6)
	// the Limit type field:
	// hard (0), soft (1), or dynamic (2)
	RtmpPeerBandwidthHard = 0
	RtmpPeerBandwidthSoft = 1
	RtmpPeerBandwidthDynamic = 2
)

/**
* the signature for packets to client.
*/
const RTMP_SIG_FMS_VER = "3,5,3,888"
const RTMP_SIG_AMF0_VER = 0
const RTMP_SIG_CLIENT_ID = "ASAICiss"

/**
* onStatus consts.
*/
const SLEVEL = "level"
const SCODE = "code"
const SDESC = "description"
const SDETAILS = "details"
const SCLIENT_ID = "clientid"
// status value
const SLEVEL_Status = "status"
// status error
const SLEVEL_Error = "error"
// code value
const SCODE_ConnectSuccess = "NetConnection.Connect.Success"
const SCODE_ConnectRejected = "NetConnection.Connect.Rejected"
const SCODE_StreamReset = "NetStream.Play.Reset"
const SCODE_StreamStart = "NetStream.Play.Start"
const SCODE_StreamPause = "NetStream.Pause.Notify"
const SCODE_StreamUnpause = "NetStream.Unpause.Notify"
const SCODE_PublishStart = "NetStream.Publish.Start"
const SCODE_DataStart = "NetStream.Data.Start"
const SCODE_UnpublishSuccess = "NetStream.Unpublish.Success"

// FMLE
const RTMP_AMF0_COMMAND_ON_FC_PUBLISH = "onFCPublish"
const RTMP_AMF0_COMMAND_ON_FC_UNPUBLISH = "onFCUnpublish"

// default stream id for response the createStream request.
const SRS_DEFAULT_SID = 1

/**
* the original request from client.
*/
// @see: SrsRequest
type RtmpRequest struct {
	/**
	* tcUrl: rtmp://request_vhost:port/app/stream
	* support pass vhost in query string, such as:
	*	rtmp://ip:port/app?vhost=request_vhost/stream
	*	rtmp://ip:port/app...vhost...request_vhost/stream
	*/
	TcUrl string
	PageUrl string
	SwfUrl string
	// enum RtmpCodecAMF0 or RtmpCodecAMF3
	ObjectEncoding int

	/**
	* parsed uri info from TcUrl and stream.
	 */
	Schema string
	Vhost string
	Port string
	App string
	Stream string
}
func NewRtmpRequest() (*RtmpRequest) {
	r := &RtmpRequest{}
	r.ObjectEncoding = RtmpCodecAMF0
	r.Port = strconv.Itoa(RtmpDefaultPort)
	return r
}

func (r *RtmpRequest) discovery_app() (err error) {
	// parse ...vhost... to ?vhost=
	var v string = r.TcUrl
	if !strings.Contains(v, "?") {
		v = strings.Replace(v, "...", "?", 1)
		v = strings.Replace(v, "...", "=", 1)
	}
	for strings.Contains(v, "...") {
		v = strings.Replace(v, "...", "&", 1)
		v = strings.Replace(v, "...", "=", 1)
	}
	r.TcUrl = v

	// parse standard rtmp url.
	var u *url.URL
	if u, err = url.Parse(r.TcUrl); err != nil {
		return
	}

	r.Schema, r.App = u.Scheme, u.Path

	r.Vhost = u.Host
	if strings.Contains(u.Host, ":") {
		host_parts := strings.Split(u.Host, ":")
		r.Vhost, r.Port = host_parts[0], host_parts[1]
	}

	// discovery vhost from query.
	query := u.Query()
	for k, _ := range query {
		if strings.ToLower(k) == "vhost" && query.Get(k) != "" {
			r.Vhost = query.Get(k)
		}
	}

	// resolve the vhost from config
	// TODO: FIXME: implements it
	// TODO: discovery the params of vhost.

	if r.Schema = strings.Trim(r.Schema, "/\n\r "); r.Schema == ""{
		return RtmpError{code:ERROR_RTMP_REQ_TCURL, desc:fmt.Sprintf("discovery schema failed. tcUrl=%v", r.TcUrl)}
	}
	if r.Vhost = strings.Trim(r.Vhost, "/\n\r "); r.Vhost == "" {
		return RtmpError{code:ERROR_RTMP_REQ_TCURL, desc:fmt.Sprintf("discovery vhost failed. tcUrl=%v", r.TcUrl)}
	}
	if r.App = strings.Trim(r.App, "/\n\r "); r.App == "" {
		return RtmpError{code:ERROR_RTMP_REQ_TCURL, desc:fmt.Sprintf("discovery app failed. tcUrl=%v", r.TcUrl)}
	}
	if r.Port = strings.Trim(r.Port, "/\n\r "); r.Port == "" {
		return RtmpError{code:ERROR_RTMP_REQ_TCURL, desc:fmt.Sprintf("discovery port failed. tcUrl=%v", r.TcUrl)}
	}

	return
}

/**
* the rtmp server interface, user can create it by func NewRtmpServer().
 */
type RtmpServer interface {
	/**
	* handshake with client, try complex handshake first, use simple if failed.
	 */
	Handshake() (err error)
	/**
	* expect client send the connect app request,
	* @param req set and parse data to the request
	 */
	ConnectApp(req *RtmpRequest) (err error)
	/**
	* set the ack size window
	* @param ack_size in bytes, for example, 2.5 * 1000 * 1000
	 */
	SetWindowAckSize(ack_size uint32) (err error)
	/**
	* set the peer bandwidth,
	* @param bandwidth in bytes, for example, 2.5 * 1000 * 1000
	* @param bw_type can be RtmpPeerBandwidthHard, RtmpPeerBandwidthSoft or RtmpPeerBandwidthDynamic
	 */
	SetPeerBandwidth(bandwidth uint32, bw_type byte) (err error)
	/**
	* response the client connect app request
	* @param req the request data genereated by ConnectApp
	* @param server_ip the ip of server to send to client, ignore if "".
	* @param extra_data the extra data to send to client, ignore if nil.
	 */
	ReponseConnectApp(req *RtmpRequest, server_ip string, extra_data map[string]string) (err error)
}
func NewRtmpServer(conn *net.TCPConn) (RtmpServer, error) {
	var err error
	r := &rtmpServer{}
	if r.protocol, err = NewRtmpProtocol(conn); err != nil {
		return r, err
	}
	return r, err
}

type rtmpServer struct {
	protocol RtmpProtocol
}

func (r *rtmpServer) Handshake() (err error) {
	// TODO: FIXME: try complex then simple handshake.
	err = r.protocol.SimpleHandshake2Client()
	return
}

func (r *rtmpServer) ConnectApp(req *RtmpRequest) (err error) {
	//var msg *RtmpMessage
	var pkt *RtmpConnectAppPacket
	if _, err = r.protocol.ExpectMessage(&pkt); err != nil {
		return
	}

	var ok bool
	if req.TcUrl, ok = pkt.CommandObject.GetPropertyString("tcUrl"); !ok {
		err = RtmpError{code:ERROR_RTMP_REQ_CONNECT, desc:"invalid request, must specifies the tcUrl."}
		return
	}
	if v, ok := pkt.CommandObject.GetPropertyString("pageUrl"); ok {
		req.PageUrl = v
	}
	if v, ok := pkt.CommandObject.GetPropertyString("swfUrl"); ok {
		req.SwfUrl = v
	}
	if v, ok := pkt.CommandObject.GetPropertyNumber("objectEncoding"); ok {
		req.ObjectEncoding = int(v)
	}

	return req.discovery_app()
}

func (r *rtmpServer) SetWindowAckSize(ack_size uint32) (err error) {
	pkt := RtmpSetWindowAckSizePacket{AcknowledgementWindowSize:ack_size}
	return r.protocol.SendPacket(&pkt, uint32(0))
}

func (r *rtmpServer) SetPeerBandwidth(bandwidth uint32, bw_type byte) (err error) {
	pkt := RtmpSetPeerBandwidthPacket{Bandwidth:bandwidth, BandwidthType:bw_type}
	return r.protocol.SendPacket(&pkt, uint32(0))
}

func (r *rtmpServer) ReponseConnectApp(req *RtmpRequest, server_ip string, extra_data map[string]string) (err error) {
	data := NewRtmpAmf0EcmaArray()
	data.Set("version", ToAmf0(RTMP_SIG_FMS_VER))
	if server_ip != "" {
		data.Set("srs_server_ip", ToAmf0(server_ip))
	}
	for k, v := range extra_data {
		data.Set(k, ToAmf0(v))
	}

	var pkt *RtmpConnectAppResPacket = NewRtmpConnectAppResPacket()
	pkt.PropsSet("fmsVer", "FMS/"+RTMP_SIG_FMS_VER).PropsSet("capabilities", float64(127)).PropsSet("mode", float64(1))
	pkt.InfoSet(SLEVEL, SLEVEL_Status).InfoSet(SCODE, SCODE_ConnectSuccess).InfoSet(SDESC, "Connection succeeded")
	pkt.InfoSet("objectEncoding", float64(req.ObjectEncoding)).InfoSet("data", data)

	return r.protocol.SendPacket(pkt, uint32(0))
}
