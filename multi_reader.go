package tease

import (
	"errors"
	"io"
)

type forwardMultiReadSeeker struct {
	readers []io.Reader
	pos     int64 // Forwarding moving position
	buf     []byte
}

func (mr *forwardMultiReadSeeker) Read(p []byte) (n int, err error) {
	for len(mr.readers) > 0 {
		// Optimization to flatten nested forwardMultiReadSeekers (Issue 13558).
		if len(mr.readers) == 1 {
			if r, ok := mr.readers[0].(*forwardMultiReadSeeker); ok {
				mr.readers = r.readers
				continue
			}
		}
		n, err = mr.readers[0].Read(p)
		mr.pos += int64(n)
		if err == io.EOF {
			// Use eofReader instead of nil to avoid nil panic
			// after performing flatten (Issue 18232).
			mr.readers[0] = eofReader{} // permit earlier GC
			mr.readers = mr.readers[1:]
		}
		if n > 0 || err != io.EOF {
			if err == io.EOF && len(mr.readers) > 0 {
				// Don't return EOF yet. More readers remain.
				err = nil
			}
			return
		}
	}
	return 0, io.EOF
}

func (mr *forwardMultiReadSeeker) Seek(offset int64, whence int) (int64, error) {
	//fmt.Println("mr.Seek() pos =", mr.pos, "off =", offset)
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = mr.pos + offset
	case io.SeekEnd:
		return 0, errors.New("ForwardMultiReadSeeker.Seek: not implemented, seek from end")
	default:
		return 0, errors.New("ForwardMultiReadSeeker.Seek: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("ForwardMultiReadSeeker.Seek: negative position")
	}

	if abs < mr.pos {
		return 0, errors.New("ForwardMultiReadSeeker.Seek: cannot go backwards!")
	}

	bl := int64(len(mr.buf))
	var np int
	var err error
	for tr := abs - mr.pos; tr > 0; tr -= bl {
		if tr < bl {
			bl = tr
		}
		np, err = mr.Read(mr.buf[:bl])
		//mr.pos += int64(np)
		if err != nil || int64(np) != bl {
			break
		}
	}

	return mr.pos, err
}

// MultiReader returns a Reader that's the logical concatenation of
// the provided input readers. They're read sequentially. Once all
// inputs have returned EOF, Read will return EOF.  If any of the readers
// return a non-nil, non-EOF error, Read will return that error.
func ForwardMultiReadSeeker(readers ...io.Reader) io.ReadSeeker {
	r := make([]io.Reader, len(readers))
	copy(r, readers)
	return &forwardMultiReadSeeker{
		readers: r,
		pos:     0,
		buf:     make([]byte, 2048),
	}
}

type eofReader struct{}

func (eofReader) Read([]byte) (int, error) {
	return 0, io.EOF
}
