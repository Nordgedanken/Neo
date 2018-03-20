package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/Nordgedanken/Morpheus/matrix"
	"github.com/Nordgedanken/Morpheus/matrix/db"
	"github.com/Nordgedanken/Morpheus/matrix/globalTypes"
	"github.com/Nordgedanken/Morpheus/matrix/messages"
	"github.com/Nordgedanken/Morpheus/matrix/rooms"
	"github.com/Nordgedanken/Morpheus/matrix/syncer"
	"github.com/Nordgedanken/Morpheus/ui/listLayouts"
	"github.com/Nordgedanken/Morpheus/utils"
	"github.com/dgraph-io/badger"
	"github.com/matrix-org/gomatrix"
	"github.com/opennota/linkify"
	"github.com/pkg/errors"
	"github.com/rhinoman/go-commonmark"
	"github.com/shibukawa/configdir"
	log "github.com/sirupsen/logrus"
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/uitools"
	"github.com/therecipe/qt/widgets"
)

var (
	MaxWorker = 3
	MaxQueue  = 20
)

// NewMainUIStruct gives you a MainUI struct with prefilled data
func NewMainUIStruct(windowWidth, windowHeight int, window *widgets.QMainWindow) (mainUIStruct *MainUI) {
	configStruct := globalTypes.Config{
		WindowWidth:  windowWidth,
		WindowHeight: windowHeight,
		Rooms:        make(map[string]*rooms.Room),
	}
	mainUIStruct = &MainUI{
		Config: configStruct,
		window: window,
	}
	return
}

// NewMainUIStructWithExistingConfig gives you a MainUI struct with prefilled data and data from a previous Config
func NewMainUIStructWithExistingConfig(configStruct globalTypes.Config, window *widgets.QMainWindow) (mainUIStruct *MainUI) {
	configStruct.Rooms = make(map[string]*rooms.Room)
	mainUIStruct = &MainUI{
		Config: configStruct,
		window: window,
	}
	return
}

// SetCli allows you to add a gomatrix.Client to your MainUI struct
func (m *MainUI) SetCli(cli *gomatrix.Client) {
	m.Cli = cli
}

// GetWidget gives you the widget of the MainUI struct
func (m *MainUI) GetWidget() (widget *widgets.QWidget) {
	widget = m.widget
	return
}

// NewUI initializes a new Main Screen
func (m *MainUI) NewUI() (err error) {

	m.Dispatcher = utils.NewDispatcher(MaxQueue, MaxWorker)
	m.Dispatcher.Run()

	m.loadChatUIDefaults()

	// Handle LogoutButton
	logoutButton := widgets.NewQPushButtonFromPointer(m.widget.FindChild("LogoutButton", core.Qt__FindChildrenRecursively).Pointer())
	logoutButton.ConnectClicked(func(_ bool) {
		LogoutErr := m.logout()
		if LogoutErr != nil {
			err = LogoutErr
			return
		}
	})

	m.initScrolls()

	go m.startSync()
	m.widget.SetSizePolicy2(widgets.QSizePolicy__Expanding, widgets.QSizePolicy__Expanding)
	m.MainWidget.SetSizePolicy2(widgets.QSizePolicy__Expanding, widgets.QSizePolicy__Expanding)

	m.RoomList.ConnectTriggerRoom(func(roomID string) {
		room := m.Rooms[roomID]

		log.Debugln("Adding New Room In Thread")
		NewRoomErr := m.RoomList.NewRoom(room, m.roomScrollArea)
		if NewRoomErr != nil {
			log.Errorln(NewRoomErr)
			return
		}
	})

	initRoomThread := core.NewQThread(nil)
	initRoomThread.ConnectRun(func() {
		m.initRoomList()
		println("initRoomList thread:", core.QThread_CurrentThread().Pointer())
	})
	initRoomThread.Start()

	var message string
	messageInput := widgets.NewQLineEditFromPointer(m.widget.FindChild("MessageInput", core.Qt__FindChildrenRecursively).Pointer())
	messageInput.ConnectTextChanged(func(value string) {
		message = value
	})

	m.window.ConnectKeyPressEvent(func(ev *gui.QKeyEvent) {
		if int(ev.Key()) == int(core.Qt__Key_Enter) || int(ev.Key()) == int(core.Qt__Key_Return) {
			go m.sendMessage(message)

			messageInput.Clear()
			ev.Accept()
		} else {
			messageInput.KeyPressEventDefault(ev)
			ev.Ignore()
		}
		return
	})

	m.RoomList.ConnectChangeRoom(func(roomID string) {
		room := m.Rooms[roomID]

		if m.CurrentRoom != room.RoomID {
			m.SetCurrentRoom(room.RoomID)

			changeRoomThread := core.NewQThread(nil)
			changeRoomThread.ConnectRun(func() {
				println("changeRoomThread thread:", core.QThread_CurrentThread().Pointer())
				count := m.MessageList.Count()
				for i := 0; i < count; i++ {
					if (i % 10) == 0 {
						m.App.ProcessEvents(core.QEventLoop__AllEvents)
					}
					widgetScroll := m.MessageList.ItemAt(i).Widget()
					widgetScroll.DeleteLater()
				}
				m.App.ProcessEvents(core.QEventLoop__AllEvents)
			})
			changeRoomThread.Start()

			m.RoomAvatar.SetPixmap(gui.NewQPixmap())
			m.MainWidget.SetWindowTitle("Morpheus - " + room.GetRoomTopic())

			room.ConnectSetAvatar(func(IMGdata []byte) {
				avatar := gui.NewQPixmap()

				str := string(IMGdata[:])
				avatar.LoadFromData(str, uint(len(str)), "", 0)
				m.RoomAvatar.SetPixmap(avatar)

				return
			})

			go room.GetRoomAvatar()

			m.RoomTitle.SetText(room.GetRoomName())
			m.RoomTopic.SetText(room.GetRoomTopic())

			log.Println("Before loadCache")
			// Ensure we count again on every Room Change
			m.MessageList.MessageCount = 0

			cacheThread := core.NewQThread(nil)
			cacheThread.ConnectRun(func() {
				m.App.ProcessEvents(core.QEventLoop__AllEvents)
				m.loadCache()
				println("cacheThread:", core.QThread_CurrentThread().Pointer())
			})
			cacheThread.Start()
		}
	})

	return
}

func (m *MainUI) initScrolls() {
	m.roomScrollArea.SetWidgetResizable(true)
	m.roomScrollArea.SetHorizontalScrollBarPolicy(core.Qt__ScrollBarAlwaysOff)
	m.roomScrollArea.SetContentsMargins(0, 0, 0, 0)
	m.roomScrollArea.SetSizeAdjustPolicy(widgets.QAbstractScrollArea__AdjustToContents)

	m.messageScrollArea.SetWidgetResizable(true)
	m.messageScrollArea.SetHorizontalScrollBarPolicy(core.Qt__ScrollBarAlwaysOff)
	m.messageScrollArea.SetContentsMargins(0, 0, 0, 0)
}

func (m *MainUI) loadChatUIDefaults() {
	m.widget = widgets.NewQWidget(nil, 0)

	var loader = uitools.NewQUiLoader(nil)
	var file = core.NewQFile2(":/qml/ui/chat.ui")

	file.Open(core.QIODevice__ReadOnly)
	m.MainWidget = loader.Load(file, m.widget)
	file.Close()

	m.messageScrollArea = widgets.NewQScrollAreaFromPointer(m.widget.FindChild("messageScroll", core.Qt__FindChildrenRecursively).Pointer())
	m.roomScrollArea = widgets.NewQScrollAreaFromPointer(m.widget.FindChild("roomScroll", core.Qt__FindChildrenRecursively).Pointer())

	m.MessageList = listLayouts.NewMessageList2(m.messageScrollArea)
	m.MessageList.Init(m.messageScrollArea)
	m.RoomList = listLayouts.NewRoomList2(m.roomScrollArea)
	m.RoomList.Init(m.roomScrollArea)

	m.RoomAvatar = widgets.NewQLabelFromPointer(m.widget.FindChild("roomAvatar", core.Qt__FindChildrenRecursively).Pointer())
	m.RoomTitle = widgets.NewQLabelFromPointer(m.widget.FindChild("RoomTitle", core.Qt__FindChildrenRecursively).Pointer())
	m.RoomTopic = widgets.NewQLabelFromPointer(m.widget.FindChild("Topic", core.Qt__FindChildrenRecursively).Pointer())

	var layout = widgets.NewQHBoxLayout()
	m.window.SetLayout(layout)
	layout.InsertWidget(0, m.MainWidget, 0, core.Qt__AlignTop|core.Qt__AlignLeft)
	layout.SetSpacing(0)
	layout.SetContentsMargins(0, 0, 0, 0)

	m.widget.ConnectResizeEvent(func(event *gui.QResizeEvent) {
		m.MainWidget.Resize(event.Size())
		event.Accept()
	})

	//Set Avatar
	avatarLogo := widgets.NewQLabelFromPointer(m.widget.FindChild("UserAvatar", core.Qt__FindChildrenRecursively).Pointer())
	go func() {
		avatar, _ := matrix.GetOwnUserAvatar(m.Cli)
		avatarLogo.SetPixmap(avatar)
	}()
}

func (m *MainUI) sendMessage(message string) (err error) {
	messageOriginal := message
	lm := linkify.Links(message)
	for _, l := range lm {
		link := message[l.Start:l.End]
		message = strings.Replace(message, link, "<a href='"+link+"'>"+link+"</a>", -1)
	}

	mardownMessage := commonmark.Md2Html(message, 0)
	if mardownMessage == message {
		_, SendErr := m.Cli.SendMessageEvent(m.CurrentRoom, "m.room.message", matrix.HTMLMessage{MsgType: "m.text", Body: messageOriginal, FormattedBody: message, Format: "org.matrix.custom.html"})
		if SendErr != nil {
			err = SendErr
			return
		}
	} else {
		_, SendErr := m.Cli.SendMessageEvent(m.CurrentRoom, "m.room.message", matrix.HTMLMessage{MsgType: "m.text", Body: message, FormattedBody: mardownMessage, Format: "org.matrix.custom.html"})
		if SendErr != nil {
			err = SendErr
			return
		}
	}
	return
}

func (m *MainUI) logout() (err error) {
	log.Infoln("Starting Logout Sequence in background")
	var wg sync.WaitGroup
	results := make(chan bool)

	wg.Add(1)
	go func(cli *gomatrix.Client, results chan<- bool) {
		defer wg.Done()
		cli.StopSync()
		_, LogoutErr := cli.Logout()
		if LogoutErr != nil {
			log.Errorln(LogoutErr)
			results <- false
		}
		cli.ClearCredentials()

		userDB, DBOpenErr := db.OpenUserDB()
		if DBOpenErr != nil {
			log.Errorln(DBOpenErr)
		}
		userDB.Close()

		configDirs := configdir.New("Nordgedanken", "Morpheus")
		filePath := filepath.ToSlash(configDirs.QueryFolders(configdir.Global)[0].Path)
		DeleteErr := os.RemoveAll(filePath + "/data/user/")
		if DeleteErr != nil {
			log.Errorln(DeleteErr)
			results <- false
		}
		db.ResetOnceUser()

		//Reset RoomList
		cacheDB, DBOpenCacheErr := db.OpenCacheDB()
		if DBOpenCacheErr != nil {
			log.Errorln(DBOpenCacheErr)
		}
		cacheDB.Update(func(txn *badger.Txn) error {
			return txn.Delete([]byte("rooms"))
		})

		results <- true
	}(m.Cli, results)

	go func() {
		wg.Wait()      // wait for each execTask to return
		close(results) // then close the results channel
	}()

	//Show LoginUI
	for result := range results {
		if result {
			m.window.DisconnectKeyPressEvent()
			m.window.DisconnectResizeEvent()
			m.widget.DisconnectResizeEvent()
			m.messageScrollArea.DisconnectResizeEvent()

			LoginUIStruct := NewLoginUIStructWithExistingConfig(m.Config, m.window)
			loginUIErr := LoginUIStruct.NewUI()
			if loginUIErr != nil {
				err = loginUIErr
				return
			}
			m.window.SetCentralWidget(LoginUIStruct.GetWidget())
		}
	}
	return
}

func (m *MainUI) startSync() (err error) {
	// Start Syncer!
	m.storage = syncer.NewMorpheusStore()

	Syncer := syncer.NewMorpheusSyncer(m.Cli.UserID, m.storage, &m.Config)

	m.Cli.Store = m.storage
	m.Cli.Syncer = Syncer
	Syncer.Store = m.storage

	Syncer.OnEventType("m.room.message", func(ev *gomatrix.Event) {
		formattedBody, _ := ev.Content["formatted_body"]
		var msg string
		msg, _ = formattedBody.(string)
		if msg == "" {
			msg, _ = ev.Body()
		}
		room := ev.RoomID
		sender := ev.Sender
		id := ev.ID
		timestamp := ev.Timestamp
		go db.CacheMessageEvents(id, sender, room, msg, timestamp)
		if room == m.CurrentRoom {
			message := messages.NewMessage()
			message.EventID = id
			message.Author = sender
			message.Message = msg
			message.Timestamp = timestamp
			message.Cli = m.Cli
			message.ConnectShowCallback(func(message *messages.Message, own bool, height, width int) {
				m.MessageList.NewMessage(message, m.messageScrollArea, own, height, width)
			})
			m.Rooms[room].AddMessage(message)

			work := utils.Job{Message: message}

			log.Infoln("sending payload  to workque")
			utils.JobQueue <- work
			log.Infoln("sent payload  to workque")
			m.MessageList.MessageCount++

			if (m.MessageList.MessageCount % 10) == 0 {
				m.App.ProcessEvents(core.QEventLoop__AllEvents)
			}
		}
		m.App.ProcessEvents(core.QEventLoop__AllEvents)
	})

	Syncer.OnEventType("m.room.name", func(ev *gomatrix.Event) {
		roomNameRaw, _ := ev.Content["name"]
		var roomName string
		roomName, _ = roomNameRaw.(string)
		evType := ev.Type
		room := ev.RoomID
		go m.Rooms[room].UpdateRoomNameByEvent(roomName, evType)
	})

	// Start Non-blocking sync
	go func() {
		log.Infoln("Start sync")
		for {
			e := m.Cli.Sync()
			if e == nil {
				break
			}
			if e != nil {
				err = e
			}
		}
	}()
	return
}

func (m *MainUI) initRoomList() (err error) {
	roomsStruct, roomsErr := rooms.GetRooms(m.Cli)
	if roomsErr != nil {
		err = roomsErr
		return
	}

	first := true
	for _, roomID := range roomsStruct {
		m.Rooms[roomID] = rooms.NewRoom()
		m.Rooms[roomID].Cli = m.Cli
		m.Rooms[roomID].RoomID = roomID
		go m.RoomList.TriggerRoom(roomID)
		m.RoomList.RoomCount++
		if (m.RoomList.RoomCount % 10) == 0 {
			m.App.ProcessEvents(core.QEventLoop__AllEvents)
		}
		if first {
			m.RoomList.ChangeRoom(roomID)
			m.App.ProcessEvents(core.QEventLoop__AllEvents)
		}
		first = false
	}

	m.App.ProcessEvents(core.QEventLoop__AllEvents)
	return
}

func (m *MainUI) loadCache() (err error) {
	log.Println("Loading cache!")

	currentRoomMem := m.Rooms[m.CurrentRoom]

	for _, v := range currentRoomMem.Messages {
		work := utils.Job{Message: v}

		log.Infoln("sending payload  to workque")
		utils.JobQueue <- work
		log.Infoln("sent payload  to workque")

		m.MessageList.MessageCount++
	}

	cacheDB, DBOpenErr := db.OpenCacheDB()
	if DBOpenErr != nil {
		log.Errorln(DBOpenErr)
		err = DBOpenErr
	}
	DBerr := cacheDB.View(func(txn *badger.Txn) error {
		MsgOpts := badger.DefaultIteratorOptions
		MsgOpts.PrefetchSize = 10
		MsgIt := txn.NewIterator(MsgOpts)

		MsgPrefix := []byte("room|" + m.CurrentRoom + "|messages")

		doneMsg := make(map[string]bool)

		var intIt int
		for MsgIt.Seek(MsgPrefix); MsgIt.ValidForPrefix(MsgPrefix); MsgIt.Next() {
			item := MsgIt.Item()
			key := item.Key()
			stringKey := fmt.Sprintf("%s", key)
			stringKeySlice := strings.Split(stringKey, "|")
			stringKeyEnd := stringKeySlice[len(stringKeySlice)-1]
			if stringKeyEnd != "id" {
				continue
			}

			value, ValueErr := item.Value()
			if ValueErr != nil {
				return ValueErr
			}
			idValue := fmt.Sprintf("%s", value)

			if !doneMsg[idValue] && (currentRoomMem.Messages[idValue] != nil) {
				// Remember we already added this message to the view
				doneMsg[idValue] = true

				// Get all Data
				senderResult, QueryErr := db.Get(txn, []byte(strings.Replace(stringKey, "|id", "|sender", -1)))
				if QueryErr != nil {
					return errors.WithMessage(QueryErr, "Key: "+strings.Replace(stringKey, "|id", "|sender", -1))
				}
				sender := fmt.Sprintf("%s", senderResult)

				msgResult, QueryErr := db.Get(txn, []byte(strings.Replace(stringKey, "|id", "|messageString", -1)))
				if QueryErr != nil {
					return errors.WithMessage(QueryErr, "Key: "+strings.Replace(stringKey, "|id", "|messageString", -1))
				}
				msg := fmt.Sprintf("%s", msgResult)

				timestampResult, QueryErr := db.Get(txn, []byte(strings.Replace(stringKey, "|id", "|timestamp", -1)))
				if QueryErr != nil {
					return errors.WithMessage(QueryErr, "Key: "+strings.Replace(stringKey, "|id", "|timestamp", -1))
				}
				timestamp := fmt.Sprintf("%s", timestampResult)

				timestampInt, ConvErr := strconv.ParseInt(timestamp, 10, 64)
				if ConvErr != nil {
					return errors.WithMessage(ConvErr, "Timestamp String: "+timestamp)
				}

				message := messages.NewMessage()
				message.EventID = idValue
				message.Author = sender
				message.Message = msg
				message.Timestamp = timestampInt
				message.Cli = m.Cli
				message.ConnectShowCallback(func(message *messages.Message, own bool, height, width int) {
					m.MessageList.NewMessage(message, m.messageScrollArea, own, height, width)
				})
				currentRoomMem.AddMessage(message)

				work := utils.Job{Message: message}

				log.Infoln("sending payload  to workque")
				utils.JobQueue <- work
				log.Infoln("sent payload  to workque")

				m.MessageList.MessageCount++

				log.Println(m.MessageList.MessageCount)

			}
			if ((m.MessageList.MessageCount % 10) == 0) || ((intIt % 10) == 0) {
				m.App.ProcessEvents(core.QEventLoop__AllEvents)
			}
			intIt++
		}
		m.App.ProcessEvents(core.QEventLoop__AllEvents)
		return nil
	})
	if DBerr != nil {
		log.Errorln("DBERR: ", DBerr)
		err = DBerr
		return
	}

	return
}
