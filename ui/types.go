package ui

import (
	"github.com/Nordgedanken/Morpheus/matrix"
	"github.com/Nordgedanken/Morpheus/matrix/db"
	"github.com/Nordgedanken/Morpheus/matrix/syncer"
	"github.com/matrix-org/gomatrix"
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"
)

// Config holds important reused information in the UI
type config struct {
	username string
	password string

	windowWidth  int
	windowHeight int

	matrixClient
}

type matrixClient struct {
	databases
	cli    *gomatrix.Client
	syncer *syncer.MorpheusSyncer
}

type databases struct {
	cacheDB db.Storer
}

// SetCurrentRoom sets the new room ID of the MainUI
func (d *databases) SetCacheDB(db db.Storer) {
	d.cacheDB = db
}

// MainUI holds information about the MainUI
type MainUI struct {
	config

	widget                  *widgets.QWidget
	widgetThread            *core.QThread
	RoomAvatar              *widgets.QLabel
	RoomTitle               *widgets.QLabel
	RoomTopic               *widgets.QLabel
	MainWidget              *widgets.QWidget
	MessageListLayout       *QVBoxLayoutWithTriggerSlot
	messageScrollArea       *widgets.QScrollArea
	messageScrollAreaThread *core.QThread

	window      *widgets.QMainWindow
	storage     *syncer.MorpheusStore
	rooms       map[string]*matrix.Room
	currentRoom string
}

// SetCurrentRoom sets the new room ID of the MainUI
func (m *MainUI) SetCurrentRoom(RoomID string) {
	m.currentRoom = RoomID
}

// LoginUI holds information about the LoginUI
type LoginUI struct {
	config
	widget *widgets.QWidget
	window *widgets.QMainWindow
}
