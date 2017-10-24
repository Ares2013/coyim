package filetransfer

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/coyim/coyim/session/access"
	sdata "github.com/coyim/coyim/session/data"
	"github.com/coyim/coyim/session/events"
	"github.com/coyim/coyim/xmpp/data"
)

var supportedFileTransferMethods = map[string]int{
	"http://jabber.org/protocol/bytestreams": 1,
	"http://jabber.org/protocol/ibb":         0,
}

var fileTransferCancelListeners = map[string]func(*recvContext){
	"http://jabber.org/protocol/ibb":         ibbWaitForCancel,
	"http://jabber.org/protocol/bytestreams": bytestreamWaitForCancel,
}

type recvContext struct {
	s       access.Session
	sid     string
	peer    string
	mime    string
	options []string
	date    string
	hash    string
	name    string
	size    int64
	desc    string
	rng     struct {
		length *int
		offset *int
	}
	destination string
	opaque      interface{}
	control     *sdata.FileTransferControl
}

func extractFileTransferOptions(f data.Form) ([]string, error) {
	if f.Type != "form" || len(f.Fields) != 1 || f.Fields[0].Var != "stream-method" || f.Fields[0].Type != "list-single" {
		return nil, fmt.Errorf("Invalid form for file transfer initiation: %#v", f)
	}
	var result []string
	for _, opt := range f.Fields[0].Options {
		result = append(result, opt.Value)
	}
	return result, nil
}

// chooseAppropriateFileTransferOptionFrom returns the file transfer option that has the highest score
// or not OK if no acceptable options are available
func chooseAppropriateFileTransferOptionFrom(options []string) (best string, ok bool) {
	bestScore := -1
	for _, opt := range options {
		score, has := supportedFileTransferMethods[opt]
		if has {
			ok = true
			if score > bestScore {
				bestScore = score
				best = opt
			}
		}

	}
	return
}

func iqResultChosenStreamMethod(opt string) data.SI {
	return data.SI{
		File: &data.File{},
		Feature: data.FeatureNegotation{
			Form: data.Form{
				Type: "submit",
				Fields: []data.FormFieldX{
					{Var: "stream-method", Values: []string{opt}},
				},
			},
		},
	}
}

func (ctx *recvContext) finalizeFileTransfer(tempName string) error {
	if err := os.Rename(tempName, ctx.destination); err != nil {
		ctx.control.ReportError(errors.New("Couldn't save final file"))
		return err
	}

	ctx.control.ReportFinished()
	removeInflightRecv(ctx.sid)

	return nil
}

func (ctx *recvContext) openDestinationTempFile() (f *os.File, err error) {
	// By creating a temp file next to the place where the real file should be saved
	// we avoid problems on linux when trying to os.Rename later - if tmp filesystem is different
	// than the destination file system. It also serves as an early permissions check.
	f, err = ioutil.TempFile(filepath.Dir(ctx.destination), filepath.Base(ctx.destination))
	if err != nil {
		ctx.opaque = nil
		ctx.control.ReportError(errors.New("Couldn't open local temporary file"))
		removeInflightRecv(ctx.sid)
	}
	return
}

func waitForFileTransferUserAcceptance(stanza *data.ClientIQ, si data.SI, acceptResult <-chan *string, ctx *recvContext) {
	result := <-acceptResult

	var error *data.ErrorReply
	if result != nil {
		opt, ok := chooseAppropriateFileTransferOptionFrom(ctx.options)
		if ok {
			setInflightRecvDestination(si.ID, *result)
			ctx.s.SendIQResult(stanza, iqResultChosenStreamMethod(opt))
			go fileTransferCancelListeners[opt](ctx)
			return
		}
		ctx.control.ReportError(errors.New("No mutually acceptable file transfer methods available"))
		error = &iqErrorBadRequest
	} else {
		error = &iqErrorForbidden
	}
	removeInflightRecv(si.ID)
	ctx.s.SendIQError(stanza, *error)
}

func registerNewFileTransfer(s access.Session, si data.SI, options []string, stanza *data.ClientIQ, f *data.File, ctl *sdata.FileTransferControl) *recvContext {
	ctx := &recvContext{
		s:       s,
		sid:     si.ID,
		mime:    si.MIMEType,
		options: options,
		date:    f.Date,
		hash:    f.Hash,
		name:    f.Name,
		size:    f.Size,
		desc:    f.Desc,
		peer:    stanza.From,
		control: ctl,
	}

	if f.Range != nil {
		ctx.rng.length = f.Range.Length
		ctx.rng.offset = f.Range.Offset
	}

	addInflightRecv(ctx)
	return ctx
}

// InitIQ is the hook function that will be called when we receive a file transfer stream initiation IQ
func InitIQ(s access.Session, stanza *data.ClientIQ, si data.SI) (ret interface{}, iqtype string, ignore bool) {
	var options []string
	var err error
	if options, err = extractFileTransferOptions(si.Feature.Form); err != nil {
		s.Warn(fmt.Sprintf("Failed to parse stream initiation: %v", err))
		return nil, "", false
	}

	f := si.File

	ctl := sdata.CreateFileTransferControl()
	ift := registerNewFileTransfer(s, si, options, stanza, f, ctl)

	acceptResult := make(chan *string)
	go waitForFileTransferUserAcceptance(stanza, si, acceptResult, ift)

	s.PublishEvent(events.FileTransfer{
		Session:          s,
		Peer:             stanza.From,
		Mime:             f.Hash,
		DateLastModified: f.Date,
		Name:             f.Name,
		Size:             f.Size,
		Description:      f.Desc,
		Answer:           acceptResult,
		Control:          ctl,
	})

	return nil, "", true
}
