package gui

import (
	"github.com/coyim/gotk3adapter/gtki"
)

type roomViewToolbar struct {
	view             gtki.Box    `gtk-widget:"room-view-toolbar"`
	roomNameLabel    gtki.Label  `gtk-widget:"room-name-label"`
	roomSubjectLabel gtki.Label  `gtk-widget:"room-subject-label"`
	roomStatusIcon   gtki.Image  `gtk-widget:"room-status-icon"`
	leaveRoomButton  gtki.Button `gtk-widget:"leave-room-button"`
}

func (v *roomView) newRoomViewToolbar() *roomViewToolbar {
	t := &roomViewToolbar{}

	t.initBuilder(v)
	t.initDefaults(v)
	t.initSubscribers(v)

	return t
}

func (t *roomViewToolbar) initBuilder(v *roomView) {
	builder := newBuilder("MUCRoomToolbar")
	panicOnDevError(builder.bindObjects(t))

	builder.ConnectSignals(map[string]interface{}{
		"on_leave_room": func() {
			t.onLeaveRoom(v)
		},
	})
}

func (t *roomViewToolbar) initDefaults(v *roomView) {
	t.leaveRoomButton.SetSensitive(v.isJoined())
	t.roomStatusIcon.SetFromPixbuf(getMUCIconPixbuf("room"))

	t.roomNameLabel.SetText(v.roomID().String())
	updateWithStyle(t.roomNameLabel, providerWithStyle("label", style{
		"font-size":   "22px",
		"font-weight": "bold",
	}))

	t.displayRoomSubject(v.room.GetSubject())
	updateWithStyle(t.roomSubjectLabel, providerWithStyle("label", style{
		"font-size":  "14px",
		"font-style": "italic",
		"color":      "#666666",
	}))
}

func (t *roomViewToolbar) initSubscribers(v *roomView) {
	v.subscribe("toolbar", func(ev roomViewEvent) {
		switch e := ev.(type) {
		case occupantSelfJoinedEvent:
			t.occupantSelfJoinedEvent()
		case subjectReceivedEvent:
			t.subjectReceivedEvent(e.subject)
		case subjectUpdatedEvent:
			t.subjectReceivedEvent(e.subject)
		}
	})
}

func (t *roomViewToolbar) occupantSelfJoinedEvent() {
	doInUIThread(func() {
		t.leaveRoomButton.SetSensitive(true)
	})
}

func (t *roomViewToolbar) subjectReceivedEvent(subject string) {
	doInUIThread(func() {
		t.displayRoomSubject(subject)
	})
}

// displayRoomSubject MUST be called from the UI thread
func (t *roomViewToolbar) displayRoomSubject(subject string) {
	t.roomSubjectLabel.SetText(subject)
	if subject == "" {
		t.roomSubjectLabel.Hide()
	} else {
		t.roomSubjectLabel.Show()
	}
}

func (t *roomViewToolbar) onLeaveRoom(v *roomView) {
	t.leaveRoomButton.SetSensitive(false)
	v.tryLeaveRoom(nil, func() {
		if v.isOpen() {
			doInUIThread(func() {
				t.leaveRoomButton.SetSensitive(true)
			})
		}
	})
}
