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
)

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
* to ensure in inserted order.
* for the FMLE will crash when AMF0Object is not ordered by inserted,
* if ordered in map, the string compare order, the FMLE will creash when
* get the response of connect app.
*/
// @see: SrsUnSortedHashtable
type RtmpAmf0UnSortedHashtable struct {
	property_index []string
	properties map[string]*RtmpAmf0Any
}
func NewRtmpAmf0UnSortedHashtable() (*RtmpAmf0UnSortedHashtable) {
	r := &RtmpAmf0UnSortedHashtable{}
	r.properties = make(map[string]*RtmpAmf0Any)
	return r
}
func (r *RtmpAmf0UnSortedHashtable) Count() (n int) {
	return len(r.properties)
}
func (r *RtmpAmf0UnSortedHashtable) Size() (n int) {
	if r.Count() <= 0 {
		return 0
	}
	for k, v := range r.properties {
		n += RtmpAmf0SizeUtf8(k)
		n += v.Size()
	}
	return
}
func (r *RtmpAmf0UnSortedHashtable) Write(codec *RtmpAmf0Codec) (err error) {
	// properties
	for _, k := range r.property_index {
		v := r.properties[k]

		if err = codec.WriteUtf8(k); err != nil {
			return
		}
		if err = v.Write(codec); err != nil {
			return
		}
	}
	return
}
func (r *RtmpAmf0UnSortedHashtable) Set(k string, v *RtmpAmf0Any) (err error) {
	if v == nil {
		err = RtmpError{code:ERROR_GO_AMF0_NIL_PROPERTY, desc:"AMF0 object property value should never be nil"}
		return
	}

	if _, ok := r.properties[k]; !ok {
		r.property_index = append(r.property_index, k)
	}
	r.properties[k] = v
	return
}
func (r *RtmpAmf0UnSortedHashtable) GetPropertyString(k string) (v string, ok bool) {
	var prop *RtmpAmf0Any
	if prop, ok = r.properties[k]; !ok {
		return
	}
	return prop.String()
}
func (r *RtmpAmf0UnSortedHashtable) GetPropertyNumber(k string) (v float64, ok bool) {
	var prop *RtmpAmf0Any
	if prop, ok = r.properties[k]; !ok {
		return
	}
	return prop.Number()
}

/**
* 2.5 Object Type
* anonymous-object-type = object-marker *(object-property)
* object-property = (UTF-8 value-type) | (UTF-8-empty object-end-marker)
*/
// @see: SrsAmf0Object
type RtmpAmf0Object struct {
	marker byte
	properties *RtmpAmf0UnSortedHashtable
}
func NewRtmpAmf0Object() (*RtmpAmf0Object) {
	r := &RtmpAmf0Object{}
	r.marker = RTMP_AMF0_Object
	r.properties = NewRtmpAmf0UnSortedHashtable()
	return r
}

func (r *RtmpAmf0Object) Size() (n int) {
	if n = r.properties.Size(); n <= 0 {
		return 0
	}

	n += 1
	n += RtmpAmf0SizeObjectEOF()
	return
}
func (r *RtmpAmf0Object) Read(codec *RtmpAmf0Codec) (err error) {
	// marker
	if !codec.stream.Requires(1) {
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 object requires 1bytes marker"}
		return
	}

	if r.marker = codec.stream.ReadByte(); r.marker != RTMP_AMF0_Object{
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 object marker invalid"}
		return
	}

	for !codec.stream.Empty() {
		// property-name: utf8 string
		var property_name string
		if property_name, err = codec.ReadUtf8(); err != nil {
			return
		}

		// property-value: any
		var property_value RtmpAmf0Any
		if err = property_value.Read(codec); err != nil {
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
func (r *RtmpAmf0Object) Write(codec *RtmpAmf0Codec) (err error) {
	// marker
	if !codec.stream.Requires(1) {
		err = RtmpError{code:ERROR_RTMP_AMF0_ENCODE, desc:"amf0 write object marker failed"}
		return
	}
	codec.stream.WriteByte(byte(RTMP_AMF0_Object))

	// properties
	if err = r.properties.Write(codec); err != nil {
		return
	}

	// object EOF
	return codec.WriteObjectEOF()
}
func (r *RtmpAmf0Object) Set(k string, v *RtmpAmf0Any) (err error) {
	return r.properties.Set(k, v)
}
func (r *RtmpAmf0Object) GetPropertyString(k string) (v string, ok bool) {
	return r.properties.GetPropertyString(k)
}
func (r *RtmpAmf0Object) GetPropertyNumber(k string) (v float64, ok bool) {
	return r.properties.GetPropertyNumber(k)
}

/**
* 2.10 ECMA Array Type
* ecma-array-type = associative-count *(object-property)
* associative-count = U32
* object-property = (UTF-8 value-type) | (UTF-8-empty object-end-marker)
*/
// @see: SrsASrsAmf0EcmaArray
type RtmpAmf0EcmaArray struct {
	marker byte
	count uint32
	properties *RtmpAmf0UnSortedHashtable
}
func NewRtmpAmf0EcmaArray() (*RtmpAmf0EcmaArray) {
	r := &RtmpAmf0EcmaArray{}
	r.marker = RTMP_AMF0_EcmaArray
	r.properties = NewRtmpAmf0UnSortedHashtable()
	return r
}

func (r *RtmpAmf0EcmaArray) Size() (n int) {
	if n = r.properties.Size(); n <= 0 {
		return 0
	}

	n += 1
	n += 4
	n += RtmpAmf0SizeObjectEOF()
	return
}
// srs_amf0_read_ecma_array
func (r *RtmpAmf0EcmaArray) Read(codec *RtmpAmf0Codec) (err error) {
	// marker
	if !codec.stream.Requires(1) {
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 EcmaArray requires 1bytes marker"}
		return
	}

	if r.marker = codec.stream.ReadByte(); r.marker != RTMP_AMF0_EcmaArray{
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 EcmaArray marker invalid"}
		return
	}

	// count
	if !codec.stream.Requires(4) {
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 read ecma_array count failed"}
		return
	}
	r.count = codec.stream.ReadUInt32()

	for !codec.stream.Empty() {
		// property-name: utf8 string
		var property_name string
		if property_name, err = codec.ReadUtf8(); err != nil {
			return
		}

		// property-value: any
		var property_value RtmpAmf0Any
		if err = property_value.Read(codec); err != nil {
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
// srs_amf0_write_ecma_array
func (r *RtmpAmf0EcmaArray) Write(codec *RtmpAmf0Codec) (err error) {
	// marker
	if !codec.stream.Requires(1) {
		err = RtmpError{code:ERROR_RTMP_AMF0_ENCODE, desc:"amf0 write EcmaArray marker failed"}
		return
	}
	codec.stream.WriteByte(byte(RTMP_AMF0_EcmaArray))

	// count
	if !codec.stream.Requires(4) {
		err = RtmpError{code:ERROR_RTMP_AMF0_ENCODE, desc:"amf0 write ecma_array count failed"}
		return
	}
	codec.stream.WriteUInt32(r.count)

	// properties
	if err = r.properties.Write(codec); err != nil {
		return
	}

	// object EOF
	return codec.WriteObjectEOF()
}
func (r *RtmpAmf0EcmaArray) Set(k string, v *RtmpAmf0Any) (err error) {
	err = r.properties.Set(k, v)
	r.count = uint32(r.properties.Count())
	return
}
func (r *RtmpAmf0EcmaArray) GetPropertyString(k string) (v string, ok bool) {
	return r.properties.GetPropertyString(k)
}
func (r *RtmpAmf0EcmaArray) GetPropertyNumber(k string) (v float64, ok bool) {
	return r.properties.GetPropertyNumber(k)
}

/**
* any amf0 value.
* 2.1 Types Overview
* value-type = number-type | boolean-type | string-type | object-type
* 		| null-marker | undefined-marker | reference-type | ecma-array-type
* 		| strict-array-type | date-type | long-string-type | xml-document-type
* 		| typed-object-type
* create any with ToAmf0(), or create a default one and Read from stream.
*/
// @see: SrsAmf0Any
type RtmpAmf0Any struct {
	Marker byte
	Value interface {}
}
func ToAmf0(v interface {}) (*RtmpAmf0Any) {
	switch t := v.(type) {
	case bool:
		return &RtmpAmf0Any{ Marker:RTMP_AMF0_Boolean, Value:t }
	case string:
		return &RtmpAmf0Any{ Marker:RTMP_AMF0_String, Value:t }
	case int:
		return &RtmpAmf0Any{ Marker:RTMP_AMF0_Number, Value:float64(t) }
	case float64:
		return &RtmpAmf0Any{ Marker:RTMP_AMF0_Number, Value:t }
	case *RtmpAmf0Object:
		return &RtmpAmf0Any{ Marker:RTMP_AMF0_Object, Value:t }
	case *RtmpAmf0EcmaArray:
		return &RtmpAmf0Any{ Marker:RTMP_AMF0_EcmaArray, Value:t }
	}
	return nil
}
func ToAmf0Null() (*RtmpAmf0Any) {
	return &RtmpAmf0Any{ Marker:RTMP_AMF0_Null }
}
func (r *RtmpAmf0Any) Size() (int) {
	switch {
	case r.Marker == RTMP_AMF0_String:
		v, _ := r.String()
		return RtmpAmf0SizeString(v)
	case r.Marker == RTMP_AMF0_Boolean:
		return RtmpAmf0SizeBoolean()
	case r.Marker == RTMP_AMF0_Number:
		return RtmpAmf0SizeNumber()
	case r.Marker == RTMP_AMF0_Null || r.Marker == RTMP_AMF0_Undefined:
		return RtmpAmf0SizeNullOrUndefined()
	case r.Marker == RTMP_AMF0_ObjectEnd:
		return RtmpAmf0SizeObjectEOF()
	case r.Marker == RTMP_AMF0_Object:
		v, _ := r.Object()
		return v.Size()
	case r.Marker == RTMP_AMF0_EcmaArray:
		v, _ := r.EcmaArray()
		return v.Size()
		// TODO: FIXME: implements it.
	}
	return 0
}
func (r *RtmpAmf0Any) Write(codec *RtmpAmf0Codec) (err error) {
	switch {
	case r.Marker == RTMP_AMF0_String:
		v, _ := r.String()
		return codec.WriteString(v)
	case r.Marker == RTMP_AMF0_Boolean:
		v, _ := r.Boolean()
		return codec.WriteBoolean(v)
	case r.Marker == RTMP_AMF0_Number:
		v, _ := r.Number()
		return codec.WriteNumber(v)
	case r.Marker == RTMP_AMF0_Null:
		return codec.WriteNull()
	case r.Marker == RTMP_AMF0_Undefined:
		return codec.WriteUndefined()
	case r.Marker == RTMP_AMF0_ObjectEnd:
		return codec.WriteObjectEOF()
	case r.Marker == RTMP_AMF0_Object:
		v, _ := r.Object()
		return v.Write(codec)
	case r.Marker == RTMP_AMF0_EcmaArray:
		v, _ := r.EcmaArray()
		return v.Write(codec)
		// TODO: FIXME: implements it.
	}
	return
}
func (r *RtmpAmf0Any) Read(codec *RtmpAmf0Codec) (err error) {
	// marker
	if !codec.stream.Requires(1) {
		err = RtmpError{code:ERROR_RTMP_AMF0_DECODE, desc:"amf0 any requires 1bytes marker"}
		return
	}
	r.Marker = codec.stream.ReadByte()
	codec.stream.Next(-1)

	switch {
	case r.Marker == RTMP_AMF0_String:
		r.Value, err = codec.ReadString()
	case r.Marker == RTMP_AMF0_Boolean:
		r.Value, err = codec.ReadBoolean()
	case r.Marker == RTMP_AMF0_Number:
		r.Value, err = codec.ReadNumber()
	case r.Marker == RTMP_AMF0_Null || r.Marker == RTMP_AMF0_Undefined || r.Marker == RTMP_AMF0_ObjectEnd:
		codec.stream.Next(1)
	case r.Marker == RTMP_AMF0_Object:
		r.Value, err = codec.ReadObject()
	case r.Marker == RTMP_AMF0_EcmaArray:
		r.Value, err = codec.ReadEcmaArray()
	// TODO: FIXME: implements it.
	default:
		err = RtmpError{code:ERROR_RTMP_AMF0_INVALID, desc:fmt.Sprintf("invalid amf0 message type. marker=%#x", r.Marker)}
	}

	return
}
func (r *RtmpAmf0Any) IsNil() (v bool) {
	return r.Value == nil
}
func (r *RtmpAmf0Any) IsObjectEof() (v bool) {
	return r.Marker == RTMP_AMF0_ObjectEnd
}
func (r *RtmpAmf0Any) Object() (v *RtmpAmf0Object, ok bool) {
	if r.Marker == RTMP_AMF0_Object {
		v, ok = r.Value.(*RtmpAmf0Object), true
	}
	return
}
func (r *RtmpAmf0Any) EcmaArray() (v *RtmpAmf0EcmaArray, ok bool) {
	if r.Marker == RTMP_AMF0_EcmaArray {
		v, ok = r.Value.(*RtmpAmf0EcmaArray), true
	}
	return
}
func (r *RtmpAmf0Any) String() (v string, ok bool) {
	if r.Marker == RTMP_AMF0_String {
		v, ok = r.Value.(string), true
	}
	return
}
func (r *RtmpAmf0Any) Number() (v float64, ok bool) {
	if r.Marker == RTMP_AMF0_Number {
		v, ok = r.Value.(float64), true
	}
	return
}
func (r *RtmpAmf0Any) Boolean() (v bool, ok bool) {
	if r.Marker == RTMP_AMF0_Boolean {
		v, ok = r.Value.(bool), true
	}
	return
}

type RtmpAmf0Codec struct {
	stream *RtmpHPBuffer
}
func NewRtmpAmf0Codec(stream *RtmpHPBuffer) (*RtmpAmf0Codec) {
	r := RtmpAmf0Codec{}
	r.stream = stream
	return &r
}

// Size
func RtmpAmf0SizeString(v string) (int) {
	return 1 + RtmpAmf0SizeUtf8(v)
}
func RtmpAmf0SizeUtf8(v string) (int) {
	return 2 + len(v)
}
func RtmpAmf0SizeNumber() (int) {
	return 1 + 8
}
func RtmpAmf0SizeNullOrUndefined() (int) {
	return 1
}
func RtmpAmf0SizeBoolean() (int) {
	return 1 + 1
}
func RtmpAmf0SizeObjectEOF() (int) {
	return 2 + 1
}

// srs_amf0_read_string
func (r *RtmpAmf0Codec) ReadString() (v string, err error) {
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

// srs_amf0_write_string
func (r *RtmpAmf0Codec) WriteString(v string) (err error) {
	// marker
	if !r.stream.Requires(1) {
		err = RtmpError{code:ERROR_RTMP_AMF0_ENCODE, desc:"amf0 write string marker failed"}
		return
	}
	r.stream.WriteByte(byte(RTMP_AMF0_String))
	return r.WriteUtf8(v)
}

// srs_amf0_write_boolean
func (r *RtmpAmf0Codec) WriteBoolean(v bool) (err error) {
	// marker
	if !r.stream.Requires(1) {
		err = RtmpError{code:ERROR_RTMP_AMF0_ENCODE, desc:"amf0 write bool marker failed"}
		return
	}
	r.stream.WriteByte(byte(RTMP_AMF0_Boolean))

	// value
	if !r.stream.Requires(1) {
		err = RtmpError{code:ERROR_RTMP_AMF0_ENCODE, desc:"amf0 write bool value failed"}
		return
	}
	if v {
		r.stream.WriteByte(byte(0x01))
	} else {
		r.stream.WriteByte(byte(0x00))
	}
	return
}

// srs_amf0_read_utf8
func (r *RtmpAmf0Codec) ReadUtf8() (v string, err error) {
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

// srs_amf0_write_utf8
func (r *RtmpAmf0Codec) WriteUtf8(v string) (err error) {
	// len
	if !r.stream.Requires(2) {
		err = RtmpError{code:ERROR_RTMP_AMF0_ENCODE, desc:"amf0 write string length failed"}
		return
	}
	r.stream.WriteUInt16(uint16(len(v)))

	// empty string
	if len(v) <= 0 {
		return
	}

	// data
	if !r.stream.Requires(len(v)) {
		err = RtmpError{code:ERROR_RTMP_AMF0_ENCODE, desc:"amf0 write string data failed"}
		return
	}
	r.stream.Write([]byte(v))
	return
}

// srs_amf0_read_number
func (r *RtmpAmf0Codec) ReadNumber() (v float64, err error) {
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

// srs_amf0_write_number
func (r *RtmpAmf0Codec) WriteNumber(v float64) (err error) {
	// marker
	if !r.stream.Requires(1) {
		err = RtmpError{code:ERROR_RTMP_AMF0_ENCODE, desc:"amf0 write number marker failed"}
		return
	}
	r.stream.WriteByte(byte(RTMP_AMF0_Number))

	// value
	if !r.stream.Requires(8) {
		err = RtmpError{code:ERROR_RTMP_AMF0_ENCODE, desc:"amf0 write number value failed"}
		return
	}
	r.stream.WriteFloat64(v)

	return
}

// srs_amf0_write_null
func (r *RtmpAmf0Codec) WriteNull() (err error) {
	// marker
	if !r.stream.Requires(1) {
		err = RtmpError{code:ERROR_RTMP_AMF0_ENCODE, desc:"amf0 write null marker failed"}
		return
	}
	r.stream.WriteByte(byte(RTMP_AMF0_Null))

	return
}

// srs_amf0_read_undefined
func (r *RtmpAmf0Codec) WriteUndefined() (err error) {
	// marker
	if !r.stream.Requires(1) {
		err = RtmpError{code:ERROR_RTMP_AMF0_ENCODE, desc:"amf0 write undefined marker failed"}
		return
	}
	r.stream.WriteByte(byte(RTMP_AMF0_Undefined))

	return
}

// srs_amf0_read_boolean
func (r *RtmpAmf0Codec) ReadBoolean() (v bool, err error) {
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
func (r *RtmpAmf0Codec) ReadObject() (v *RtmpAmf0Object, err error) {
	// value
	v = NewRtmpAmf0Object()
	return v, v.Read(r)
}

// srs_amf0_read_ecma_array
func (r *RtmpAmf0Codec) ReadEcmaArray() (v *RtmpAmf0EcmaArray, err error) {
	// value
	v = NewRtmpAmf0EcmaArray()
	return v, v.Read(r)
}

// srs_amf0_write_object
func (r *RtmpAmf0Codec) WriteObject(v *RtmpAmf0Object) (err error) {
	return v.Write(r)
}

// srs_amf0_read_ecma_array
func (r *RtmpAmf0Codec) WriteEcmaArray(v *RtmpAmf0EcmaArray) (err error) {
	return v.Write(r)
}

// srs_amf0_write_object_eof
func (r *RtmpAmf0Codec) WriteObjectEOF() (err error) {
	// value
	if !r.stream.Requires(2) {
		err = RtmpError{code:ERROR_RTMP_AMF0_ENCODE, desc:"amf0 write object eof value failed"}
		return
	}
	r.stream.WriteUInt16(uint16(0))

	// marker
	if !r.stream.Requires(1) {
		err = RtmpError{code:ERROR_RTMP_AMF0_ENCODE, desc:"amf0 write object eof marker failed"}
		return
	}
	r.stream.WriteByte(byte(RTMP_AMF0_ObjectEnd))
	return
}
