package filetransfer

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/coyim/coyim/session/access"
	"github.com/coyim/coyim/xmpp/data"
)

func init() {
	registerSendFileTransferMethod("http://jabber.org/protocol/ibb", ibbSendDo, ibbSendCurrentlyValid)
}

func ibbSendCurrentlyValid(string, access.Session) bool {
	return true
}

const ibbDefaultBlockSize = 4096

func ibbSendDo(ctx *sendContext) {
	ibbSendDoWithBlockSize(ctx, ibbDefaultBlockSize)
}

func ibbSendDoWithBlockSize(ctx *sendContext, blocksize int) {
	nonblockIQ(ctx.s, ctx.peer, "set", data.IBBOpen{
		BlockSize: ibbDefaultBlockSize,
		Sid:       ctx.sid,
		Stanza:    "iq",
	}, nil, func(*data.ClientIQ) {
		go ibbSendStartTransfer(ctx, blocksize)
	}, func(ciq *data.ClientIQ, e error) {
		if ciq != nil &&
			ciq.Type == "error" &&
			ciq.Error.Type == "modify" &&
			ciq.Error.Condition.XMLName.Local == "resource-constraint" &&
			ciq.Error.Condition.XMLName.Space == "urn:ietf:params:xml:ns:xmpp-stanzas" {
			ibbSendDoWithBlockSize(ctx, blocksize/2)
			return
		}
		ctx.onError(e)
	})
}

func ibbReadAndWrite(ctx *sendContext) io.ReadCloser {
	f, err := os.Open(ctx.file)
	if err != nil {
		ctx.onError(err)
		return nil
	}

	r, w := io.Pipe()
	w2 := base64.NewEncoder(base64.StdEncoding, w)
	w3, beforeFinish := ctx.enc.wrapForSending(w2)

	go func() {
		_, err = io.Copy(w3, f)
		if err != nil && err != errLocalCancel {
			ctx.onError(err)
		}

		beforeFinish()

		_ = w3.Close()
	}()

	return r
}

func ibbSendChunk(ctx *sendContext, r io.ReadCloser, buffer []byte, seq uint16) bool {
	if ctx.weWantToCancel {
		_, _, _ = ctx.s.Conn().SendIQ(ctx.peer, "set", data.IBBClose{Sid: ctx.sid})
		removeInflightSend(ctx)
		return false
	} else if ctx.theyWantToCancel {
		ctx.control.CloseAll()
		removeInflightSend(ctx)
		return false
	}

	n, err := r.Read(buffer)
	if n > 0 {
		rpl, _, e := ctx.s.Conn().SendIQ(ctx.peer, "set", data.IBBData{
			Sid:      ctx.sid,
			Sequence: seq,
			Base64:   string(buffer[:n]),
		})
		if e != nil {
			ctx.onError(e)
			return false
		}
		ctx.onUpdate(n)

		go trackResultOfSend(ctx, rpl)
	}
	if err == io.EOF {
		closeAndIgnore(r)
		_, _, _ = ctx.s.Conn().SendIQ(ctx.peer, "set", data.IBBClose{Sid: ctx.sid})
		ctx.onFinish()
		return false
	} else if err != nil {
		ctx.onError(err)
		return false
	}

	return true
}

func trackResultOfSend(ctx *sendContext, reply <-chan data.Stanza) {
	select {
	case r := <-reply:
		switch ciq := r.Value.(type) {
		case *data.ClientIQ:
			if ciq.Type == "result" {
				return
			}
		}
		ctx.s.Info(fmt.Sprintf("Received unhappy response to IBB data sent: %#v", r))
		ctx.theyWantToCancel = true
	case <-time.After(time.Minute * 5):
		// Ignore timeout
	}
}

func ibbScheduleNextSend(ctx *sendContext, r io.ReadCloser, buffer []byte, seq uint16) bool {
	time.AfterFunc(time.Duration(200)*time.Millisecond, func() {
		ibbSendChunks(ctx, r, buffer, seq)
	})

	return true
}

func ibbSendChunks(ctx *sendContext, r io.ReadCloser, buffer []byte, seq uint16) {
	// The seq variable can wrap around here - THAT IS ON PURPOSE
	// See XEP-0047 for details
	_ = ibbSendChunk(ctx, r, buffer, seq) &&
		ibbSendChunk(ctx, r, buffer, seq+1) &&
		ibbSendChunk(ctx, r, buffer, seq+2) &&
		ibbSendChunk(ctx, r, buffer, seq+3) &&
		ibbSendChunk(ctx, r, buffer, seq+4) &&
		ibbScheduleNextSend(ctx, r, buffer, seq+5)
}

func ibbSendStartTransfer(ctx *sendContext, blockSize int) {
	seq := uint16(0)
	buffer := make([]byte, blockSize)
	r := ibbReadAndWrite(ctx)
	if r == nil {
		return
	}
	ibbSendChunks(ctx, r, buffer, seq)
}

func ibbReceivedClose(ctx *sendContext) {
	ctx.theyWantToCancel = true
}
