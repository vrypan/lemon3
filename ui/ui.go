package ui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vrypan/lemon3/enclosure"
	"github.com/vrypan/lemon3/farcaster"
	"github.com/vrypan/lemon3/fctools"
	"github.com/vrypan/lemon3/ipfsServer"
)

type CastAdd struct {
	Cast *farcaster.Message
}
type FollowAdd struct {
	Fid uint64
}
type FollowRemove struct {
	Fid uint64
}
type ErrorEvent struct {
	Error error
}
type SystemError struct {
	Error error
}
type MainModel struct {
	Ipfs            *ipfsServer.IpfsServer
	Hub             *fctools.FarcasterHub
	Fid             uint64
	PrivateKey      []byte
	state           string
	casts           []*CastItem
	follows         map[uint64]bool
	listModel       listModel
	filePickerModel filePickerModel
	formModel       formModel
	Err             error
	statusMsg       string
	width           int
	height          int
	chFcMsg         chan *farcaster.Message
	chErrs          chan error
	ChEncl          chan *enclosure.Enclosure
	everyoneMode    bool
}

// State constants
const (
	stateList       = "list"
	stateFilePicker = "filePicker"
	stateForm       = "form"
)

// Init initializes the Bubble Tea model
func (m *MainModel) Init() tea.Cmd {
	m.chFcMsg = make(chan *farcaster.Message, 1)
	m.chErrs = make(chan error, 1)
	m.follows = make(map[uint64]bool)
	m.listModel = newListModel(m)
	m.filePickerModel = newFilePickerModel(m, ".")
	m.formModel = newFormModel(m)
	m.state = stateList
	//defer close(m.chMsgData)
	//defer close(m.chErrs)
	m.listenForErrors()
	if errCmd := m.SubscribeToFcEvents(); errCmd != nil {
		return errCmd
	}
	if m.Fid > 0 { // Get our own casts
		m.UpdateWithOldCasts(m.Fid)
	}
	return m.listenForNewFcEvents()

	// return nil
}

// Update handles messages and user input
func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Declare cmd without assignment yet
	var cmd tea.Cmd

	// Handle window size changes at the top level
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
	}

	switch msg := msg.(type) {
	case CastAdd:
		m.statusMsg = fmt.Sprintf("NewCast: %s", msg.Cast.Data.GetCastAddBody().Text)
		m.casts = append(m.casts, NewCastItem(msg.Cast).Render(m.Hub))
		return m, m.listenForNewFcEvents()
	case FollowAdd:
		m.statusMsg = fmt.Sprintf("FollowAdd %d", msg.Fid)
		m.follows[msg.Fid] = true
		return m, m.listenForNewFcEvents()
	case FollowRemove:
		m.statusMsg = fmt.Sprintf("FollowRemove %d", msg.Fid)
		m.follows[msg.Fid] = false
		return m, m.listenForNewFcEvents()
	case ErrorEvent:
		m.Err = msg.Error
		return m, m.listenForErrors()
	case SystemError:
		m.Err = msg.Error
		return m, tea.Quit
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Err = nil
			return m, tea.Quit
		}
	}
	// Delegate message handling based on current state
	switch m.state {
	case stateList:
		newListModel, newCmd := m.listModel.Update(msg)
		m.listModel = newListModel.(listModel)
		cmd = newCmd
	case stateFilePicker:
		newFilePickerModel, newCmd := m.filePickerModel.Update(msg)
		m.filePickerModel = newFilePickerModel.(filePickerModel)
		cmd = newCmd
	case stateForm:
		newFormModel, newCmd := m.formModel.Update(msg)
		m.formModel = newFormModel.(formModel)
		cmd = newCmd
	}

	return m, cmd
}

// View renders the current view
func (m MainModel) View() string {
	// Render the view based on the current state
	switch m.state {
	case stateList:
		return m.listModel.View()
	case stateFilePicker:
		return m.filePickerModel.View()
	case stateForm:
		return m.formModel.View()
	default:
		return "Unknown state"
	}
}

// ShowFilePicker switches to the file picker state
func (m *MainModel) ShowFilePicker() {
	currentDir, err := os.Getwd()
	if err != nil {
		m.Err = err
		return
	}

	// Create a new file picker with the current dimensions from the parent model
	m.filePickerModel = newFilePickerModel(m, currentDir)

	// Pass the current width and height from the parent model
	if m.width > 0 && m.height > 0 {
		m.filePickerModel.width = m.width
		m.filePickerModel.height = m.height

		// Set maxVisible based on the height
		m.filePickerModel.maxVisible = m.height - 6
		if m.filePickerModel.maxVisible < 5 {
			m.filePickerModel.maxVisible = 5 // Minimum
		}
	}

	m.state = stateFilePicker
}

// ShowForm switches to the form state with the selected file
func (m *MainModel) ShowForm(filePath string) {
	m.formModel = newFormModel(m)
	m.formModel.selectedFile = filePath
	m.state = stateForm
}

// ShowList switches back to the list state
func (m *MainModel) ShowList() {
	m.state = stateList
}

func (m *MainModel) AddCast(cast *farcaster.Message) {
	m.casts = append(m.casts, NewCastItem(cast).Render(m.Hub))
}

func (m *MainModel) UpdateWithOldCasts(fid uint64) {
	casts, err := m.Hub.GetCastsByFid(fid, nil, 1000)
	if err != nil {
		fmt.Println("Error fetching casts:", err)
		return
	}
	for _, c := range casts.Messages {
		body := c.GetData().GetCastAddBody()
		if body.Embeds != nil && strings.HasPrefix(body.Embeds[0].GetUrl(), "enclosure+ipfs://") {
			m.casts = append(m.casts, NewCastItem(c).Render(m.Hub))
		}
	}
}

func (m *MainModel) listenForNewFcEvents() tea.Cmd {
	return func() tea.Msg {
		msg := <-m.chFcMsg
		switch msg.Data.Type {
		case farcaster.MessageType_MESSAGE_TYPE_CAST_ADD:
			return CastAdd{Cast: msg}
		case farcaster.MessageType_MESSAGE_TYPE_LINK_ADD:
			return FollowAdd{Fid: msg.Data.GetLinkBody().GetTargetFid()}
		case farcaster.MessageType_MESSAGE_TYPE_LINK_REMOVE:
			return FollowRemove{Fid: msg.Data.GetLinkBody().GetTargetFid()}
		default:
			return nil
		}
	}
}
func (m *MainModel) listenForErrors() tea.Cmd {
	return func() tea.Msg {
		msg := <-m.chErrs
		return ErrorEvent{Error: msg}
	}
}
