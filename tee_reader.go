package tease

import (
	"errors"
	"io"
)

// TeeReader returns a Reader that writes to w what it reads from r.
// All reads from r performed through it are matched with
// corresponding writes to w. There is no internal buffering -
// the write must complete before the read completes.
// Any error encountered while writing is reported as a read error.
func TeeReadSeeker(r io.Reader, w io.Writer) *teeReadSeeker {
	return &teeReadSeeker{
		r:   r,
		w:   w,
		buf: make([]byte, 2048),
	}
}

type teeReadSeeker struct {
	r   io.Reader
	w   io.Writer
	buf []byte
	pos int64
}

func (t *teeReadSeeker) Read(p []byte) (n int, err error) {
	n, err = t.r.Read(p)
	if n > 0 {
		if n, err := t.w.Write(p[:n]); err != nil {
			return n, err
		}
	}
	return
}

func (mr *teeReadSeeker) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = mr.pos + offset
	case io.SeekEnd:
		return 0, errors.New("TeeReadSeeker.Seek: not implemented, seek from end")
	default:
		return 0, errors.New("TeeReadSeeker.Seek: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("TeeReadSeeker.Seek: negative position")
	}
	if abs < mr.pos {
		return 0, errors.New("TeeReadSeeker.Seek: cannot go backwards!")
	}

	bl := int64(len(mr.buf))
	var np int
	var err error
	for tr := abs - mr.pos; tr > 0; tr -= bl {
		if tr < bl {
			bl = tr
		}
		np, err = mr.Read(mr.buf[:bl])
		mr.pos += int64(np)
		if err != nil || int64(np) != bl {
			break
		}
	}

	return mr.pos, err
}
