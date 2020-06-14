package filetransfer

import (
	"fmt"

	"github.com/coyim/coyim/session/access"
	sdata "github.com/coyim/coyim/session/data"
	"github.com/coyim/coyim/session/events"
	"github.com/coyim/coyim/xmpp/data"
	"github.com/coyim/coyim/xmpp/jid"
)

// InitIQ is the hook function that will be called when we receive a file or directory transfer stream initiation IQ
func InitIQ(s access.Session, stanza *data.ClientIQ, si data.SI) (ret interface{}, iqtype string, ignore bool) {
	isDir := false
	//isEnc := false
	switch si.Profile {
	case dirTransferProfile:
		isDir = true
	case encryptedTransferProfile:
		//isEnc = true
		isDir = si.EncryptedData.Type == "directory"
	}
	//isEnc = isEnc && config.EncryptedFileTransferEnabled

	var options []string
	var err error
	if options, err = extractFileTransferOptions(si.Feature.Form); err != nil {
		s.Warn(fmt.Sprintf("Failed to parse stream initiation: %v", err))
		return nil, "", false
	}

	ctl := sdata.CreateFileTransferControl()

	// TODO: here until the end we need to figure out what to do with encryption
	ctx := registerNewFileTransfer(s, si, options, stanza, ctl, isDir)

	acceptResult := make(chan *string)
	go waitForFileTransferUserAcceptance(stanza, si, acceptResult, ctx)

	peer, ok := jid.Parse(stanza.From).(jid.WithResource)
	if !ok {
		s.Warn(fmt.Sprintf("Stanza sender doesn't contain resource - this shouldn't happen: %v", stanza.From))
		return nil, "", false
	}

	s.PublishEvent(events.FileTransfer{
		Peer:             peer,
		Mime:             si.File.Hash,
		DateLastModified: si.File.Date,
		Name:             si.File.Name,
		Size:             si.File.Size,
		Description:      si.File.Desc,
		Answer:           acceptResult,
		Control:          ctl,
		IsDirectory:      isDir,
	})

	return nil, "", true
}
