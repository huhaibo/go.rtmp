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

// AMF0 marker
const RTMP_AMF0_Number = 0x00
const RTMP_AMF0_Boolean = 0x01
const RTMP_AMF0_String = 0x02
const RTMP_AMF0_Object = 0x03
const RTMP_AMF0_MovieClip = 0x04 // reserved, not supported
const RTMP_AMF0_Null = 0x05
const RTMP_AMF0_Undefined = 0x06
const RTMP_AMF0_Reference = 0x07
const RTMP_AMF0_EcmaArray = 0x08
const RTMP_AMF0_ObjectEnd = 0x09
const RTMP_AMF0_StrictArray = 0x0A
const RTMP_AMF0_Date = 0x0B
const RTMP_AMF0_LongString = 0x0C
const RTMP_AMF0_UnSupported = 0x0D
const RTMP_AMF0_RecordSet = 0x0E // reserved, not supported
const RTMP_AMF0_XmlDocument = 0x0F
const RTMP_AMF0_TypedObject = 0x10
// AVM+ object is the AMF3 object.
const RTMP_AMF0_AVMplusObject = 0x11
// origin array whos data takes the same form as LengthValueBytes
const RTMP_AMF0_OriginStrictArray = 0x20

// User defined
const RTMP_AMF0_Invalid = 0x3F

type RtmpAmf0Object struct {
}

type Amf0Codec interface {
	ReadString() (v string, err error)
}
func NewAmf0Codec(stream RtmpStream) (Amf0Codec) {
	r := amf0Codec{}
	r.stream = stream
	return &r
}
type amf0Codec struct {
	stream RtmpStream
}

// srs_amf0_read_string
func (r *amf0Codec) ReadString() (v string, err error) {
	// marker
	if !r.stream.Requires(1) {
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 string requires 1bytes marker"}
		return
	}

	if marker := r.stream.ReadByte(); marker != RTMP_AMF0_String {
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 string marker invalid"}
		return
	}

	v, err = r.ReadUtf8()
	return
}

// srs_amf0_read_utf8
func (r *amf0Codec) ReadUtf8() (v string, err error) {
	// len
	if !r.stream.Requires(2) {
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 utf8 len requires 2bytes"}
		return
	}
	len := r.stream.ReadUInt16()

	// empty string
	if len <= 0 {
		return
	}

	// data
	if !r.stream.Requires(int(len)) {
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 utf8 data requires more bytes"}
		return
	}
	v = r.stream.ReadString(int(len))

	// support utf8-1 only
	// 1.3.1 Strings and UTF-8
	// UTF8-1 = %x00-7F
	for _, ch := range v {
		if (ch & 0x80) != 0 {
			// ignored. only support utf8-1, 0x00-0x7F
			//err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"only support utf8-1, 0x00-0x7F"}
			//return
		}
	}

	return
}
