package gui

import (
	"errors"
	"fmt"

	"github.com/coyim/coyim/session/events"
	"github.com/coyim/coyim/session/muc"
	"github.com/coyim/coyim/xmpp/jid"
)

func (u *gtkUI) getRoomView(rid jid.Bare, account *account) (*roomView, *muc.Room, error) {
	room, exists := account.roomManager.GetRoom(rid)
	if !exists {
		return nil, nil, errors.New("The rooms doesn't exists")
	}

	rv := room.Opaque.(*roomView)

	return rv, room, nil
}

func (u *gtkUI) roomOcuppantJoinedOn(account *account, ev events.MUCOccupantJoined) {
	rid := jid.Parse(ev.From).(jid.Bare)
	rv, room, err := u.getRoomView(rid, account)
	if err != nil {
		account.log.WithError(err)
	}
	// Updating the room occupant in the room manager
	from := fmt.Sprintf("%s/%s", ev.From, ev.Nickname)
	fjid := jid.Parse(from).(jid.WithResource)
	rjid := jid.Parse(ev.Jid).(jid.WithResource)
	room.Roster().UpdatePresence(fjid, "", ev.Affiliation, ev.Role, "", ev.Status, "Room Joined", rjid)
	rv.roomOcuppantJoinedOn(err)
}

func (u *gtkUI) roomOccupantUpdatedOn(account *account, ev events.MUCOccupantUpdated) {
	//TODO: Implements the actions to do when a Occupant presence is received
	u.log.Info("roomOccupantUpdatedOn")
}

func (u *gtkUI) roomOcuppantJoinFailedOn(account *account, ev events.MUCError) {
	from := jid.Parse(ev.EventInfo.From)
	ridwr, nickname := from.PotentialSplit()
	rid := jid.Parse(ridwr.String()).(jid.Bare)
	rv, _, err := u.getRoomView(rid, account)
	if err != nil {
		account.log.WithError(err)
	}
	errorMessage := fmt.Sprintf("Nickname conflict, can't join to the room using \"%s\"", nickname)
	rv.roomOcuppantJoinedOn(errors.New(errorMessage))
}
