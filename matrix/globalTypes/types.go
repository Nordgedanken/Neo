package globalTypes

import (
	"github.com/Nordgedanken/Morpheus/matrix/db"
	"github.com/Nordgedanken/Morpheus/matrix/rooms"
	"github.com/Nordgedanken/Morpheus/ui/listLayouts"
	"github.com/matrix-org/gomatrix"
)

// Config holds important reused information in the UI
type Config struct {
	Localpart string
	Password  string
	Server    string

	WindowWidth  int
	WindowHeight int

	Rooms       map[string]*rooms.Room
	CurrentRoom string

	MessageList *listLayouts.MessageList
	RoomList    *listLayouts.RoomList

	matrixClient
}

type matrixClient struct {
	databases
	Cli *gomatrix.Client
}

// GetCli returns the Matrix Client
func (mc *matrixClient) GetCli() *gomatrix.Client {
	return mc.Cli
}

type databases struct {
	CacheDB db.Storer
}

// SetCurrentRoom sets the new room ID of the MainUI
func (d *databases) SetCacheDB(db db.Storer) {
	d.CacheDB = db
}