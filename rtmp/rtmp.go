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
)

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

type RtmpServer interface {
	Handshake() (err error)
	ConnectApp(req *RtmpRequest) (err error)
	SetWindowAckSize(ack_size uint32) (err error)
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
	err = r.protocol.SendMessage(&pkt, uint32(0))
	return
}
