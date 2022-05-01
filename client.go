package tease

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// Wrapper for the net.Conn connection allowing client protocol testing.
type Client struct {
	// constant
	conn    net.Conn
	isPiped bool
	err     error

	// Maximum number of bytes to be buffered.  If the reader or writer tries to
	// read past this the connection is terminated.
	MaxBuffer int

	// input/output
	rawInput  []byte // raw input buffer
	inputCnt  int
	rawOutput []byte // raw output buffer
	outputCnt int
	mu        sync.Mutex
}

// Create a new teaser in client mode.  In client mode new outgoing connections
// can be replayed over different endpoints.  Returning packets are read to
// verify success.
func NewClient(conn net.Conn) *Client {
	return &Client{
		conn:      conn,
		MaxBuffer: 1024,
	}
}

// Change the client connection and send out the write buffer.
func (c *Client) SetNewConn(conn net.Conn) (err error) {
	if c.conn != nil {
		c.conn.Close()
	}
	c.conn = conn
	c.outputCnt, err = c.conn.Write(c.rawOutput)
	return
}

func (c *Client) String() string {
	return fmt.Sprintf("tease_client{pipe: %v, read: %d, readQue: %d, write: %d, writeQue: %d}",
		c.isPiped, c.inputCnt, len(c.rawInput), c.outputCnt, len(c.rawOutput))
}

func (c *Client) Replay() error {
	if c.isPiped {
		// We are already connected, no reply allowed
		return errAlreadyPipe
	}

	// wipe buffers in the alternate direction
	c.rawInput = []byte{}

	// reset counters
	c.inputCnt, c.outputCnt = 0, 0
	if c.err == errClosed {
		c.err = nil
	}
	return nil
}

// Pipe the connections together, basically pipe the reads and writes
func (c *Client) Pipe() (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// trim input buffer
	if c.inputCnt > 0 {
		c.rawInput = c.rawInput[c.inputCnt:]
	}

	// flush output buffer
	if len(c.rawOutput) > 0 && c.outputCnt < len(c.rawOutput) {
		_, err = c.conn.Write(c.rawOutput[c.outputCnt-1:])
		c.err = err
	}

	// reset counters
	c.inputCnt, c.outputCnt = 0, 0

	// mark as connected
	c.isPiped = true

	return
}

// ReadByte reads and returns the next byte from the input or any error
// encountered. If ReadByte returns an error, no input byte was consumed, and
// the returned byte value is undefined.
func (c *Client) ReadByte() (byte, error) {
	buf1 := []byte{0}
	n, err := c.Read(buf1)
	if n == 1 && err == nil {
		return buf1[0], nil
	}
	if err != nil {
		return byte(0), err
	}
	return byte(0), errors.New("EOF")
}

// Read reads data from the connection.
// Read can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetReadDeadline.
func (c *Client) Read(b []byte) (n int, err error) {
	n, err = c.conn.Read(b)
	return
}

// Write writes data to the connection.
// Write can be made to time out and return an error after a fixed
// time limit; see SetDeadline and SetWriteDeadline.
func (c *Client) Write(b []byte) (n int, err error) {
	// Short circuit if in pipe mode
	if c.isPiped {
		n, err = c.conn.Write(b)
		return
	}
	n, err = c.write(b)
	return
}

func (c *Client) write(b []byte) (n int, err error) {
	// If we are in an error state, give up
	if err != nil {
		return 0, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Mind limits
	if len(c.rawOutput)+len(b) > c.MaxBuffer {
		err, c.err = errMaxBuffer, errMaxBuffer
		c.conn.Close()
		return
	}

	// add writes to buffer
	c.rawOutput = append(c.rawOutput, b...)
	n = len(b)

	n, err = c.conn.Write(b)
	c.outputCnt += n
	return
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *Client) Close() error {
	if c.isPiped {
		// Only allow piped connections to be closed
		return c.conn.Close()
	}

	c.err = errClosed
	return nil
}

// LocalAddr returns the local network address.
func (c *Client) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *Client) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail instead of blocking. The deadline applies to all future
// and pending I/O, not just the immediately following call to
// Read or Write. After a deadline has been exceeded, the
// connection can be refreshed by setting a deadline in the future.
//
// If the deadline is exceeded a call to Read or Write or to other
// I/O methods will return an error that wraps os.ErrDeadlineExceeded.
// This can be tested using errors.Is(err, os.ErrDeadlineExceeded).
// The error's Timeout method will return true, but note that there
// are other possible errors for which the Timeout method will
// return true even if the deadline has not been exceeded.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
func (c *Client) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (c *Client) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (c *Client) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
