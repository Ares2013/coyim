package xmpp

import (
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"gopkg.in/check.v1"
)

type mockConnIOReaderWriter struct {
	read      []byte
	readIndex int
	write     []byte
	errCount  int
	err       error

	calledClose int

	lock sync.Mutex
}

func (iom *mockConnIOReaderWriter) CalledClose() bool {
	iom.lock.Lock()
	defer iom.lock.Unlock()
	return iom.calledClose > 0
}

func (iom *mockConnIOReaderWriter) Written() []byte {
	iom.lock.Lock()
	defer iom.lock.Unlock()

	var res []byte
	l := len(iom.write)
	res = make([]byte, l)
	copy(res, iom.write)
	return res
}

func (iom *mockConnIOReaderWriter) Read(p []byte) (n int, err error) {
	iom.lock.Lock()
	defer iom.lock.Unlock()

	if iom.readIndex >= len(iom.read) {
		return 0, io.EOF
	}
	i := copy(p, iom.read[iom.readIndex:])
	iom.readIndex += i
	var e error
	if iom.errCount == 0 {
		e = iom.err
	}
	iom.errCount--
	return i, e
}

func (iom *mockConnIOReaderWriter) Write(p []byte) (n int, err error) {
	iom.lock.Lock()
	defer iom.lock.Unlock()

	iom.write = append(iom.write, p...)
	var e error
	if iom.errCount == 0 {
		e = iom.err
	}
	iom.errCount--
	return len(p), e
}

func (iom *mockConnIOReaderWriter) Close() error {
	iom.lock.Lock()
	defer iom.lock.Unlock()

	iom.calledClose++
	return nil
}

type mockMultiConnIOReaderWriter struct {
	read      [][]byte
	readIndex int
	write     []byte
}

func (iom *mockMultiConnIOReaderWriter) Read(p []byte) (n int, err error) {
	if iom.readIndex >= len(iom.read) {
		return 0, io.EOF
	}
	i := copy(p, iom.read[iom.readIndex])
	iom.readIndex++
	return i, nil
}

func (iom *mockMultiConnIOReaderWriter) Write(p []byte) (n int, err error) {
	iom.write = append(iom.write, p...)
	return len(p), nil
}

type fullMockedConn struct {
	rw io.ReadWriter
}

func (c *fullMockedConn) Read(b []byte) (n int, err error) {
	return c.rw.Read(b)
}

func (c *fullMockedConn) Write(b []byte) (n int, err error) {
	return c.rw.Write(b)
}

func (c *fullMockedConn) Close() error {
	return nil
}

func (c *fullMockedConn) LocalAddr() net.Addr {
	return nil
}

func (c *fullMockedConn) RemoteAddr() net.Addr {
	return nil
}

func (c *fullMockedConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *fullMockedConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *fullMockedConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type fixedRandReader struct {
	data []string
	at   int
}

func fixedRand(data []string) io.Reader {
	return &fixedRandReader{data, 0}
}

func bytesFromHex(s string) []byte {
	val, _ := hex.DecodeString(s)
	return val
}

func (frr *fixedRandReader) Read(p []byte) (n int, err error) {
	if frr.at < len(frr.data) {
		plainBytes := bytesFromHex(frr.data[frr.at])
		frr.at++
		n = copy(p, plainBytes)
		return
	}
	return 0, io.EOF
}

type dialCall func(string, string) (c net.Conn, e error)
type dialCallExp struct {
	f      dialCall
	called bool
}

type mockProxy struct {
	called int
	calls  []dialCallExp
	sync.Mutex
}

func (p *mockProxy) Dial(network, addr string) (net.Conn, error) {
	if len(p.calls)-1 < p.called {
		return nil, fmt.Errorf("unexpected call to Dial: %s, %s", network, addr)
	}

	p.Lock()
	defer p.Unlock()

	fn := p.calls[p.called]
	p.called = p.called + 1

	fn.called = true
	return fn.f(network, addr)
}

func (p *mockProxy) Expects(f dialCall) {
	p.Lock()
	defer p.Unlock()

	if p.calls == nil {
		p.calls = []dialCallExp{}
	}

	p.calls = append(p.calls, dialCallExp{f: f})
}

var MatchesExpectations check.Checker = &allExpectations{
	&check.CheckerInfo{Name: "IsNil", Params: []string{"value"}},
}

type allExpectations struct {
	*check.CheckerInfo
}

func (checker *allExpectations) Check(params []interface{}, names []string) (result bool, error string) {
	p := params[0].(*mockProxy)

	if p.called != len(p.calls) {
		return false, fmt.Sprintf("expected: %d calls, got: %d", len(p.calls), p.called)
	}

	return true, ""
}
