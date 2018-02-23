package ui

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/Nordgedanken/Morpheus/matrix/globalTypes"
	log "github.com/sirupsen/logrus"
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/uitools"
	"github.com/therecipe/qt/widgets"
)

const redBorder = "border: 1px solid red"

// NewRegUIStruct gives you a RegUI struct with prefilled data
func NewRegUIStruct(windowWidth, windowHeight int, window *widgets.QMainWindow) (regUIStruct *RegUI) {
	configStruct := globalTypes.Config{
		WindowWidth:  windowWidth,
		WindowHeight: windowHeight,
	}
	regUIStruct = &RegUI{
		Config: configStruct,
		window: window,
	}
	return
}

// NewLoginUIStructWithExistingConfig gives you a LoginUI struct with prefilled data and data from a previous Config
func NewRegUIStructWithExistingConfig(configStruct globalTypes.Config, window *widgets.QMainWindow) (regUIStruct *RegUI) {
	regUIStruct = &RegUI{
		Config: configStruct,
		window: window,
	}
	return
}

// GetWidget gives you the widget of the LoginUI struct
func (r *RegUI) GetWidget() (widget *widgets.QWidget) {
	widget = r.widget
	return
}

// NewUI initializes a new login Screen
func (r *RegUI) NewUI() (err error) {
	r.widget = widgets.NewQWidget(nil, 0)

	var loader = uitools.NewQUiLoader(nil)
	var file = core.NewQFile2(":/qml/ui/register.ui")

	file.Open(core.QIODevice__ReadOnly)
	r.RegWidget = loader.Load(file, r.widget)
	file.Close()

	// UsernameInput
	usernameInput := widgets.NewQLineEditFromPointer(r.widget.FindChild("UsernameInput", core.Qt__FindChildrenRecursively).Pointer())

	// PasswordInput
	passwordInput := widgets.NewQLineEditFromPointer(r.widget.FindChild("PasswordInput", core.Qt__FindChildrenRecursively).Pointer())

	// PasswordConfirmInput
	//passwordConfirmInput := widgets.NewQLineEditFromPointer(r.widget.FindChild("PasswordConfirmInput", core.Qt__FindChildrenRecursively).Pointer())

	// ServerDropdown
	serverDropdown := widgets.NewQComboBoxFromPointer(r.widget.FindChild("ServerChooserDropdown", core.Qt__FindChildrenRecursively).Pointer())

	// registerButton
	registerButton := widgets.NewQPushButtonFromPointer(r.widget.FindChild("RegisterButton", core.Qt__FindChildrenRecursively).Pointer())

	// loginButton
	loginButton := widgets.NewQPushButtonFromPointer(r.widget.FindChild("loginButton", core.Qt__FindChildrenRecursively).Pointer())

	loginButton.ConnectClicked(func(_ bool) {
		loginUIStruct := NewLoginUIStructWithExistingConfig(r.Config, r.window)
		logUIErr := loginUIStruct.NewUI()
		if logUIErr != nil {
			err = logUIErr
			return
		}
		r.window.SetCentralWidget(loginUIStruct.GetWidget())
		r.window.Resize(r.widget.Size())
	})

	var helloMatrixRespErr error
	r.helloMatrixResp, helloMatrixRespErr = getHelloMatrixList()
	if helloMatrixRespErr != nil {
		log.Println(helloMatrixRespErr)
		err = helloMatrixRespErr
		return
	}

	hostnames := convertHelloMatrixRespToNameSlice(r.helloMatrixResp)
	serverDropdown.AddItems(hostnames)

	var layout = widgets.NewQHBoxLayout()
	r.window.SetLayout(layout)
	layout.InsertWidget(0, r.RegWidget, 0, core.Qt__AlignTop|core.Qt__AlignLeft)
	layout.SetSpacing(0)
	layout.SetContentsMargins(0, 0, 0, 0)
	r.widget.SetSizePolicy2(widgets.QSizePolicy__Expanding, widgets.QSizePolicy__Expanding)
	r.RegWidget.SetSizePolicy2(widgets.QSizePolicy__Expanding, widgets.QSizePolicy__Expanding)

	r.widget.ConnectResizeEvent(func(event *gui.QResizeEvent) {
		r.RegWidget.Resize(event.Size())
		event.Accept()
	})

	usernameInput.ConnectTextChanged(func(value string) {
		if usernameInput.StyleSheet() == redBorder {
			usernameInput.SetStyleSheet("")
		}
		r.Localpart = value
	})

	passwordInput.ConnectTextChanged(func(value string) {
		if passwordInput.StyleSheet() == redBorder {
			passwordInput.SetStyleSheet("")
		}
		r.Password = value
	})

	registerButton.ConnectClicked(func(_ bool) {
		if r.Localpart != "" && r.Password != "" {
			r.Server = serverDropdown.CurrentText()
			RegisterErr := r.register()
			if RegisterErr != nil {
				err = RegisterErr
				return
			}
		} else {
			passwordInput.SetStyleSheet(redBorder)
		}
	})

	usernameInput.ConnectKeyPressEvent(func(ev *gui.QKeyEvent) {
		if int(ev.Key()) == int(core.Qt__Key_Enter) || int(ev.Key()) == int(core.Qt__Key_Return) {
			if r.Password != "" {
				r.Server = serverDropdown.CurrentText()
				RegisterErr := r.register()
				if RegisterErr != nil {
					err = RegisterErr
					return
				}

				usernameInput.Clear()
				ev.Accept()
			} else {
				passwordInput.SetStyleSheet(redBorder)
				ev.Ignore()
			}
		} else {
			usernameInput.KeyPressEventDefault(ev)
			ev.Ignore()
		}
	})

	passwordInput.ConnectKeyPressEvent(func(ev *gui.QKeyEvent) {
		if int(ev.Key()) == int(core.Qt__Key_Enter) || int(ev.Key()) == int(core.Qt__Key_Return) {
			if r.Localpart != "" {
				r.Server = serverDropdown.CurrentText()
				RegisterErr := r.register()
				if RegisterErr != nil {
					err = RegisterErr
					return
				}

				passwordInput.Clear()
				ev.Accept()
			} else {
				usernameInput.SetStyleSheet(redBorder)
				ev.Ignore()
			}
		} else {
			passwordInput.KeyPressEventDefault(ev)
			ev.Ignore()
		}
	})

	r.RegWidget.SetWindowTitle("Morpheus - Register")

	return
}

func (r *RegUI) register() error {
	return nil
}

func getHelloMatrixList() (resp helloMatrixResp, err error) {
	var httpClient = &http.Client{Timeout: 10 * time.Second}

	url := "https://www.hello-matrix.net/public_servers.php?format=json&only_public=true"

	r, RespErr := httpClient.Get(url)
	if RespErr != nil {
		err = RespErr
		return
	}
	defer r.Body.Close()

	decodeErr := json.NewDecoder(r.Body).Decode(&resp)
	if decodeErr != nil {
		err = decodeErr
		return
	}

	return
}

func convertHelloMatrixRespToNameSlice(resp helloMatrixResp) (hostnames []string) {
	hostnames = append(hostnames, "Select a Server")

	sort.Slice(resp, func(i, j int) bool {
		return resp[i].LastResponseTime < resp[i].LastResponseTime
	})
	for _, v := range resp {
		hostnames = append(hostnames, v.Hostname)
	}

	return
}
