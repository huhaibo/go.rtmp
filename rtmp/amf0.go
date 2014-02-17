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

import "fmt"

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

/**
* 2.5 Object Type
* anonymous-object-type = object-marker *(object-property)
* object-property = (UTF-8 value-type) | (UTF-8-empty object-end-marker)
*/
// @see: SrsAmf0Object
type RtmpAmf0Object struct {
	properties map[string]*RtmpAmf0Any
}
func NewRtmpAmf0Object() (*RtmpAmf0Object) {
	r := &RtmpAmf0Object{}
	r.properties = make(map[string]*RtmpAmf0Any)
	return r
}

func (r *RtmpAmf0Object) Read(codec RtmpAmf0Codec, s RtmpStream) (err error) {
	for !s.Empty() {
		// property-name: utf8 string
		var property_name string
		if property_name, err = codec.ReadUtf8(); err != nil {
			return
		}

		// property-value: any
		var property_value RtmpAmf0Any
		if err = property_value.Read(codec, s); err != nil {
			return
		}

		// AMF0 Object EOF.
		if len(property_name) <= 0 || property_value.IsNil() || property_value.IsObjectEof() {
			break
		}

		// add property
		if err = r.Set(property_name, &property_value); err != nil {
			return
		}
	}
	return
}
func (r *RtmpAmf0Object) Set(k string, v *RtmpAmf0Any) (err error) {
	if v == nil {
		err = RtmpError{code:ERROR_RTMP_AMF0_NIL_PROPERTY, desc:"AMF0 object property value should never be nil"}
		return
	}
	r.properties[k] = v
	return
}
func (r *RtmpAmf0Object) GetPropertyString(k string) (v string, ok bool) {
	var prop *RtmpAmf0Any
	if prop, ok = r.properties[k]; !ok {
		return
	}
	return prop.String()
}
func (r *RtmpAmf0Object) GetPropertyNumber(k string) (v float64, ok bool) {
	var prop *RtmpAmf0Any
	if prop, ok = r.properties[k]; !ok {
		return
	}
	return prop.Number()
}

/**
* any amf0 value.
* 2.1 Types Overview
* value-type = number-type | boolean-type | string-type | object-type
* 		| null-marker | undefined-marker | reference-type | ecma-array-type
* 		| strict-array-type | date-type | long-string-type | xml-document-type
* 		| typed-object-type
*/
// @see: SrsAmf0Any
type RtmpAmf0Any struct {
	Maker byte
	Value interface {}
}

func (r *RtmpAmf0Any) Read(codec RtmpAmf0Codec, s RtmpStream) (err error) {
	// marker
	if !s.Requires(1) {
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 any requires 1bytes marker"}
		return
	}
	r.Maker = s.TopByte()

	switch {
	case r.Maker == RTMP_AMF0_String:
		r.Value, err = codec.ReadString()
	case r.Maker == RTMP_AMF0_Boolean:
		r.Value, err = codec.ReadBoolean()
	case r.Maker == RTMP_AMF0_Number:
		r.Value, err = codec.ReadNumber()
	case r.Maker == RTMP_AMF0_Null || r.Maker == RTMP_AMF0_Undefined || r.Maker == RTMP_AMF0_ObjectEnd:
		s.Next(1)
	case r.Maker == RTMP_AMF0_Object:
		r.Value, err = codec.ReadObject()
	// TODO: FIXME: implements it.
	default:
		err = RtmpError{code:ERROR_RTMP_AMF0_INVALID, desc:fmt.Sprintf("invalid amf0 message type. marker=%#x", r.Maker)}
	}

	return
}
func (r *RtmpAmf0Any) IsNil() (v bool) {
	return r.Value == nil
}
func (r *RtmpAmf0Any) IsObjectEof() (v bool) {
	return r.Maker == RTMP_AMF0_ObjectEnd
}
func (r *RtmpAmf0Any) String() (v string, ok bool) {
	if r.Maker == RTMP_AMF0_String {
		v, ok = r.Value.(string), true
	}
	return
}
func (r *RtmpAmf0Any) Number() (v float64, ok bool) {
	if r.Maker == RTMP_AMF0_Number {
		v, ok = r.Value.(float64), true
	}
	return
}
func (r *RtmpAmf0Any) Boolean() (v bool, ok bool) {
	if r.Maker == RTMP_AMF0_Boolean {
		v, ok = r.Value.(bool), true
	}
	return
}

type RtmpAmf0Codec interface {
	ReadString() (v string, err error)
	ReadBoolean() (v bool, err error)
	ReadUtf8() (v string, err error)
	ReadNumber() (v float64, err error)
	ReadObject() (v *RtmpAmf0Object, err error)
}
func NewRtmpAmf0Codec(stream RtmpStream) (RtmpAmf0Codec) {
	r := rtmpAmf0Codec{}
	r.stream = stream
	return &r
}

type rtmpAmf0Codec struct {
	stream RtmpStream
}

// srs_amf0_read_string
func (r *rtmpAmf0Codec) ReadString() (v string, err error) {
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
func (r *rtmpAmf0Codec) ReadUtf8() (v string, err error) {
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

// srs_amf0_read_number
func (r *rtmpAmf0Codec) ReadNumber() (v float64, err error) {
	// marker
	if !r.stream.Requires(1) {
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 number requires 1bytes marker"}
		return
	}

	if marker := r.stream.ReadByte(); marker != RTMP_AMF0_Number{
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 number marker invalid"}
		return
	}

	// value
	if !r.stream.Requires(8) {
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 number requires 8bytes value"}
		return
	}
	v = r.stream.ReadFloat64()

	return
}

// srs_amf0_read_boolean
func (r *rtmpAmf0Codec) ReadBoolean() (v bool, err error) {
	// marker
	if !r.stream.Requires(1) {
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 bool requires 1bytes marker"}
		return
	}

	if marker := r.stream.ReadByte(); marker != RTMP_AMF0_Boolean{
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 bool marker invalid"}
		return
	}

	// value
	if !r.stream.Requires(1) {
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 bool requires 8bytes value"}
		return
	}

	if r.stream.ReadByte() == 0 {
		v = false
	} else {
		v = true
	}

	return
}

// srs_amf0_read_object
func (r *rtmpAmf0Codec) ReadObject() (v *RtmpAmf0Object, err error) {
	// marker
	if !r.stream.Requires(1) {
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 object requires 1bytes marker"}
		return
	}

	if marker := r.stream.ReadByte(); marker != RTMP_AMF0_Object{
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 object marker invalid"}
		return
	}

	// value
	v = NewRtmpAmf0Object()
	if err = v.Read(r, r.stream); err != nil {
		return
	}

	return
}
