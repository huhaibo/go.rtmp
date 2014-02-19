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

/**
* high performance bytes buffer, read and write from zero.
 */
const HPBufferMaxSize = 1024 * 1024
type HPBuffer struct {
	buf []byte
	off int
}
func NewHPBuffer(b []byte) (*HPBuffer) {
	r := &HPBuffer{}
	r.buf = b
	return r
}
func (r *HPBuffer) Bytes() []byte { return r.buf[r.off:] }
func (r *HPBuffer) String() string {
	if r == nil {
		return "<nil>"
	}
	return string(r.buf[r.off:])
}
func (r *HPBuffer) Reset() { r.off = 0 }
func (r *HPBuffer) Len() (int) { return len(r.buf) - r.off }
func (r *HPBuffer) Append(b []byte) (n int, err error) {
	// TODO: FIXME: return err
	n = len(b)
	r.buf = append(r.buf, b...)
	return
}
func (r *HPBuffer) Truncate() (err error) {
	// TODO: FIXME: return err
	r.buf = r.buf[r.off:len(r.buf)]
	r.off = 0

	// shrink if too large
	if cap(r.buf) - len(r.buf) > HPBufferMaxSize {
		v := make([]byte, len(r.buf))
		copy(v, r.buf)
		r.buf = v
	}
	return
}
func (r *HPBuffer) Next(n int) (b []byte) {
	if n > 0 {
		b = r.buf[r.off:r.off+n]
	} else {
		b = r.buf[r.off+n:r.off]
	}
	r.off += n
	return
}
func (r *HPBuffer) Read(b []byte) (n int, err error) {
	// TODO: FIXME: return err
	n = len(b)
	copy(b, r.buf[r.off:r.off+n])
	r.off += n
	return
}
func (r *HPBuffer) ReadByte() (v byte, err error) {
	// TODO: FIXME: return err
	v = r.buf[r.off]
	r.off += 1
	return
}
func (r *HPBuffer) Write(b []byte) (n int, err error) {
	// TODO: FIXME: return err
	n = len(b)
	copy(r.buf[r.off:r.off+n], b)
	r.off += n
	return
}
func (r *HPBuffer) WriteByte(b byte) (err error) {
	// TODO: FIXME: return err
	r.buf[r.off] = b
	r.off += 1
	return
}
