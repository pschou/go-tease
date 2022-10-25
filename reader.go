package tease

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

type Reader struct {
	r     io.Reader
	buf   *bytes.Buffer
	r_tee *teeReadSeeker
	r_mr  io.ReadSeeker
	pos   int64
	pipe  bool
	//reset *func() error
}

func NewReader(r io.Reader) *Reader {
	buf := bytes.NewBuffer([]byte{})
	return &Reader{
		r:     r,
		buf:   buf,
		r_tee: TeeReadSeeker(r, buf),
		pipe:  false,
	}
}

func (c *Reader) Close() {
	if c.buf != nil {
		c.buf.Reset()
	}
	c.pos = 0
	c.r = nil
	c.r_tee = nil
	c.r_mr = nil
}

//func (c *Reader) ResetFunc(f func() error) {
//	c.reset = &f
//}

/*func (c *Reader) Close() {
	c.r = nil
	c.r_pos = 0
	c.buf.Reset()
}*/
func (c *Reader) Stats() {
	fmt.Println("pos =", c.pos, "r =", c.r, "r_tee =", c.r_tee, "r_mr =", c.r_mr, "buf len =", c.buf.Len())
}
func (c *Reader) Pipe() {
	if c.pipe {
		return
	}
	//fmt.Println("Pipe called, pos =", c.pos)
	c.pipe = true
	r := ForwardMultiReadSeeker(interface{}(c.buf).(io.Reader), c.r)
	r.Seek(c.pos, io.SeekStart)
	c.r_mr = r
}

func (c *Reader) Seek(offset int64, whence int) (int64, error) {
	if c.pipe { // inline the seeker provided by the pipe
		return c.r_mr.Seek(offset, whence)
	}
	return c.seek(offset, whence)
}

// Seek without pipe
func (c *Reader) seek(offset int64, whence int) (n int64, err error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = c.pos + offset
	case io.SeekEnd:
		return 0, errors.New("Reader.Seek: not implemented, seek from end")
	default:
		return 0, errors.New("Reader.Seek: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("Reader.Seek: negative position")
	}

	if abs > int64(c.buf.Len()) {
		n, err = c.r_tee.Seek(abs, io.SeekStart)
		c.pos = n
		return c.pos, err
	}
	c.pos = abs
	return c.pos, err
}

func (c *Reader) Read(b []byte) (n int, err error) {
	if c.pipe {
		return c.r_mr.Read(b)
	}
	n, err = c.ReadAt(b, c.pos)
	//if c.pipe && err == io.EOF {
	//	c.eof = true
	//}
	return
}
func (c *Reader) ReadByte() (byte, error) {
	oneByte := []byte{0}
	_, err := c.Read(oneByte)
	return oneByte[0], err
}

func (c *Reader) ReadAt(p []byte, off int64) (int, error) {
	//fmt.Println("readat called", len(p), "off", off, "pos", c.pos)
	if c.pipe {
		if off < c.pos {
			return 0, errors.New("Reader already piped, cannot go backwards!")
			/*if c.reset == nil {
			} /*else {
				reset := *c.reset
				err = reset()
				if err != nil {
					return
				}
				c.pos, c.r_pos = 0, 0
			}*/
		}

		n, err := c.r_mr.Seek(off, io.SeekStart)
		if n != off || err != nil {
			return 0, err
		}
		return c.r_mr.Read(p)
	}

	// Seek filling buffer
	n, err := c.seek(off+int64(len(p)), io.SeekStart)
	if n <= off {
		return 0, err
	}

	// Read off the slice
	bufBytes := c.buf.Bytes()
	copied := copy(p, bufBytes[off:])
	//fmt.Println("...copied", n)
	return copied, err
}
