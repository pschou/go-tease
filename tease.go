// Written by Paul Schou (github.com/pschou/go-tease)
// MIT License, see LICENSE for details
package tease

import "errors"

var (
	errClosed      = errors.New("tease: invalid use of closed connection")
	errHasWriten   = errors.New("tease: cannot read after write without pipe mode")
	errAlreadyPipe = errors.New("tease: connection already in pipe mode")
	errMaxBuffer   = errors.New("tease: request exceeded MaxBuffer, closing connection")
)
