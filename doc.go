/*
GoLang Tease - The package for protocol detection and testing.

When detecting the protocol, a special tool need to be able to test incoming
packets against a variety of decoders is needed.  The teaser is the answer.  It
mimics an either a server or client connection, while offering net.Conn like
interface.  This way one can do a read off a connection stream, and the teaser
will prevent the reader from closing or sending on this connection.  If the
protocol reader is successful, the programmer can call the Pipe() function to
flatten out the connection teaser into a simple input/output passthrough.  If
the protocol reader is unsuccessful, the programmer can call Replay(), which
resets the reads to the beginning of the input stream and allows a different
protocol tester to be applied.

*/
package tease
