package gui

import (
	"errors"
	"log"
	"sync"

	"github.com/coyim/coyim/session/events"
	"github.com/coyim/coyim/ui"
	"github.com/coyim/coyim/xmpp"
	"github.com/coyim/coyim/xmpp/data"
	"github.com/coyim/coyim/xmpp/interfaces"
	"github.com/coyim/coyim/xmpp/utils"
	"github.com/coyim/gotk3adapter/glibi"
	"github.com/coyim/gotk3adapter/gtki"
)

type addChatView struct {
	accountManager *accountManager

	gtki.Dialog `gtk-widget:"add-chat-dialog"`

	model   gtki.ListStore `gtk-widget:"accounts-model"`
	account gtki.ComboBox  `gtk-widget:"accounts"`
	service gtki.Entry     `gtk-widget:"service"`
	room    gtki.Entry     `gtk-widget:"room"`
	handle  gtki.Entry     `gtk-widget:"handle"`
}

func newChatView(accountManager *accountManager) gtki.Dialog {
	view := &addChatView{
		accountManager: accountManager,
	}

	builder := newBuilder("AddChat")
	err := builder.bindObjects(view)
	if err != nil {
		panic(err)
	}

	builder.ConnectSignals(map[string]interface{}{
		"join_room_handler": view.joinRoomHandler,
		"cancel_handler":    view.Destroy,
	})

	doInUIThread(view.populateModel)

	return view
}

func (v *addChatView) populateModel() {
	accs := v.accountManager.getAllConnectedAccounts()
	for _, acc := range accs {
		iter := v.model.Append()
		v.model.SetValue(iter, 0, acc.session.GetConfig().Account)
		v.model.SetValue(iter, 1, acc.session.GetConfig().ID())
	}

	if len(accs) > 0 {
		v.account.SetActive(0)
	}
}

//TODO: This is repeated on AddAccount logic, for example.
func (v *addChatView) getAccount() (*account, bool) {
	iter, err := v.account.GetActiveIter()
	if err != nil {
		return nil, false
	}

	val, err := v.model.GetValue(iter, 1)
	if err != nil {
		return nil, false
	}

	id, err := val.GetString()
	if err != nil {
		return nil, false
	}

	return v.accountManager.getAccountByID(id)
}

func (v *addChatView) validateForm() (*account, *data.Occupant, error) {
	account, ok := v.getAccount()
	if !ok {
		return nil, nil, errors.New("could not find account")
	}

	//TODO: If service is empty, should get it from account's JID
	service, err := v.service.GetText()
	if err != nil {
		return nil, nil, err
	}

	room, err := v.room.GetText()
	if err != nil {
		return nil, nil, err
	}

	handle, err := v.handle.GetText()
	if err != nil {
		return nil, nil, err
	}

	//TODO: VALIDATE!

	occ := &data.Occupant{
		Room: data.Room{
			ID:      room,
			Service: service,
		},
		Handle: handle,
	}

	return account, occ, nil
}

func (v *addChatView) joinRoomHandler() {
	account, occupant, err := v.validateForm()
	if err != nil {
		//TODO: show error
		return
	}

	chatRoom := newChatRoomView(account, occupant)
	if parent, err := v.GetTransientFor(); err == nil {
		chatRoom.SetTransientFor(parent)
	}
	v.Destroy()

	account.session.Subscribe(chatRoom.eventsChan)
	chatRoom.openWindow()
}

func (u *gtkUI) addChatRoom() {
	//pass message and presence channels
	view := newChatView(u.accountManager)
	view.SetTransientFor(u.window)
	view.Show()
}

type roomOccupant struct {
	Role        string
	Affiliation string
}

type chatRoomView struct {
	gtki.Window `gtk-widget:"muc-window"`
	entry       gtki.Entry `gtk-widget:"text-box"`

	historyMutex  sync.Mutex
	historyBuffer gtki.TextBuffer     `gtk-widget:"chat-buffer"`
	historyScroll gtki.ScrolledWindow `gtk-widget:"chat-box"`

	occupantsList struct {
		sync.Mutex

		dirty bool
		m     map[string]*roomOccupant
	}
	occupantsView  gtki.TreeView  `gtk-widget:"occupants-view"`
	occupantsModel gtki.ListStore `gtk-widget:"occupants"`

	eventsChan chan interface{}
	chat       interfaces.Chat
	occupant   *data.Occupant
}

func newChatRoomView(account *account, occupant *data.Occupant) *chatRoomView {
	conn := account.session.Conn()
	if conn == nil {
		return nil
	}

	builder := newBuilder("MUCMockup")
	mockup := &chatRoomView{
		chat:     conn.GetChatContext(),
		occupant: occupant,

		//TODO: This could go somewhere else (account maybe?)
		eventsChan: make(chan interface{}),
	}

	mockup.occupantsList.m = make(map[string]*roomOccupant, 5)

	err := builder.bindObjects(mockup)
	if err != nil {
		panic(err)
	}

	builder.ConnectSignals(map[string]interface{}{
		"send_message_handler":             mockup.onSendMessage,
		"scroll_history_to_bottom_handler": mockup.scrollHistoryToBottom,

		//TODO: A closed window will leave the room
		//Probably not what we want for the final version
		"leave_room_handler": mockup.leaveRoom,
	})

	mockup.SetTitle(occupant.Room.JID())

	return mockup
}

func (v *chatRoomView) showDebugInfo() {
	//TODO Remove this. It is only for debugging
	if v.occupant == nil {
		return
	}

	if !v.chat.CheckForSupport(v.occupant.Service) {
		log.Println("No support to MUC")
	} else {
		log.Println("MUC is supported")
	}

	rooms, err := v.chat.QueryRooms(v.occupant.Service)
	if err != nil {
		log.Println(err)
	}

	log.Printf("%s has rooms:", v.occupant.Service)
	for _, i := range rooms {
		log.Printf("- %s\t%s", i.Jid, i.Name)
	}

	response, err := v.chat.QueryRoomInformation(v.occupant.Room.JID())
	if err != nil {
		log.Println("Error to query room information")
		log.Println(err)
	}

	log.Printf("RoomInfo: %#v", response)
}

func (v *chatRoomView) openWindow() {
	//TODO: show error
	go v.chat.EnterRoom(v.occupant)

	go v.watchEvents(v.eventsChan)

	//TODO: remove me
	//go v.showDebugInfo()

	v.Show()
}

func (v *chatRoomView) leaveRoom() {
	v.chat.LeaveRoom(v.occupant)
	close(v.eventsChan)
	v.eventsChan = nil
}

func (v *chatRoomView) watchEvents(evs <-chan interface{}) {
	for {
		v.redrawOccupantsList()

		ev, ok := <-evs
		if !ok {
			return
		}

		//TODO: Disable controls when the session disconnects

		switch e := ev.(type) {
		case events.ChatPresence:
			from := xmpp.ParseJID(e.ClientPresence.From)
			if from.Bare() != v.occupant.Room.JID() {
				log.Println("muc: presence not for this room. %#v", e.ClientPresence)
				continue
			}

			v.updatePresence(e.ClientPresence)
		case events.ChatMessage:
			destination := xmpp.ParseJID(e.ClientMessage.From)
			if v.occupant.Room.ID != destination.LocalPart ||
				v.occupant.Room.Service != destination.DomainPart {
				continue
			}

			//TODO: ignore messages not for this room
			doInUIThread(func() {
				v.displayReceivedMessage(&e)
			})
		default:
			//Ignore
			log.Printf("chat view got event: %#v", e)
		}
	}
}

func (v *chatRoomView) updatePresence(presence *data.ClientPresence) {
	v.occupantsList.Lock()
	defer v.occupantsList.Unlock()

	log.Println("muc: update presence status for: %#v", presence)
	v.occupantsList.dirty = true
	v.occupantsList.m[presence.From] = &roomOccupant{} //TODO: parse from presence <x />
}

func (v *chatRoomView) redrawOccupantsList() {
	if !v.occupantsList.dirty {
		return
	}

	v.occupantsList.Lock()
	defer v.occupantsList.Unlock()
	v.occupantsList.dirty = false

	doInUIThread(func() {
		//TODO
		//See: https://en.wikibooks.org/wiki/GTK%2B_By_Example/Tree_View/Tree_Models#Speed_Issues_when_Adding_a_Lot_of_Rows
		//v.occupantsView.SetModel(nil)
		v.occupantsModel.Clear()

		for jid, occupant := range v.occupantsList.m {
			iter := v.occupantsModel.Append()
			//Set other values from occupant
			_ = occupant
			v.occupantsModel.SetValue(iter, 0, xmpp.ParseJID(jid).ResourcePart)
		}

		//TODO
		//v.occupantsList.SetModel(v.users)
	})
}

func (v *chatRoomView) displayReceivedMessage(message *events.ChatMessage) {
	v.appendToHistory(message)
	//TODO: maybe notify?
}

func (v *chatRoomView) appendToHistory(message *events.ChatMessage) {
	v.historyMutex.Lock()
	defer v.historyMutex.Unlock()

	start := v.historyBuffer.GetCharCount()
	if start != 0 {
		insertAtEnd(v.historyBuffer, "\n")
	}

	sent := sentMessage{
		//TODO: Why both?
		message:         message.Body,
		strippedMessage: ui.StripSomeHTML([]byte(message.Body)),

		from:      utils.ResourceFromJid(message.From),
		to:        message.To,
		timestamp: message.When,
	}

	//TODO: use attention?
	entries, _ := sent.Tagged()

	insertTimestamp(v.historyBuffer, message.When)
	for _, e := range entries {
		insertEntry(v.historyBuffer, e)
	}

	v.scrollHistoryToBottom()
}

func (v *chatRoomView) scrollHistoryToBottom() {
	scrollToBottom(v.historyScroll)
}

func (v *chatRoomView) connectOrSendMessage(msg string) {
	//TODO: append message to the message view
	v.chat.SendChatMessage(msg, &v.occupant.Room)
}

func (v *chatRoomView) onSendMessage(_ glibi.Object) {
	//TODO: Why cant I use entry as gtki.Entry?
	//TODO: File a bug againt gotkadapter

	msg, err := v.entry.GetText()
	if err != nil {
		return
	}

	v.entry.SetText("")

	go v.connectOrSendMessage(msg)
}

func (u *gtkUI) openMUCMockup() {
	accounts := u.getAllConnectedAccounts()
	mockup := newChatRoomView(accounts[0], nil)
	mockup.SetTransientFor(u.window)
	mockup.Show()
}
