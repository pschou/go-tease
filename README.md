# GoLang Tease - The package for protocol detection and testing.


A Go library for protocol testing / analyzing.  This way incoming/outgoing stream connections can be tested against a variety of decoders before the connection is fully engaged (buffered and replay capabilities)

With this tool a developer is able to test incoming
packets against a variety of decoders as needed.  It
mimics an either a server or client connection, while offering a net.Conn type
interface.  This way one can do a read off a connection stream, and the teaser
will prevent the reader from closing or sending replies on this connection.  

When a
protocol reader is successful, the programmer can call the Pipe() function to
flatten out the connection, reducing teaser into a simple input/output passthrough.  If
the protocol reader is unsuccessful, the programmer can call Replay(), which
resets the reads to the beginning of the input stream and allows a different
protocol tester to be applied.

# Example Server usage
```
func (server *Server) handleConnection(rawConn net.Conn) {
	// Load up a new teaser
  teaseConn := tease.NewServer(rawConn)
  var conn net.Conn  // Handle for the actual connection to use

  // Read off 10 bytes from the incoming packet for inspection
  initDat := make([]byte, 10)
  _, err := teaseConn.Read(initDat)

  if err != nil { // Return if error reading
    return
  }
  teaseConn.Replay() // Reset the teaser to byte 0

  ...
  // Example test for TLS handshake
  if conn == nil && initDat[0] == 0x16 {
     ...
     // Verify TLS here and set conn
     teaseConn.Pipe()  // Connect the teaser to the input
     conn = tls.Server(teaseConn, server.TLSConfig)  // Go ahead with the wrapper
  }

  // Example test for JSON data blob
  if conn == nil && initDat[0] == '{' {
     ...
     teaseConn.Pipe()  // Connect the teaser to the input
     conn = teaseConn // Go ahead without a wrapper
  }

  // When none of the tests passed
  if conn == nil {
     return
  }
  ...

  // Work with the `conn` handle for further reads and writes

}
```


