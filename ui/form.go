package ui

import (
	"fmt"
	"mime"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vrypan/lemon3/enclosure"
	pb "github.com/vrypan/lemon3/farcaster"
)

// formModel manages the form for adding a new enclosure
type formModel struct {
	parent       *MainModel
	selectedFile string
	inputs       []textinput.Model
	focusIndex   int
	width        int
	height       int
	err          error
	isSubmitting bool
}

// formFieldTitle is the index of the title field
const formFieldFileName = 0

// formFieldDescription is the index of the description field
const formFieldDescription = 1

// newFormModel creates a new form model
func newFormModel(parent *MainModel) formModel {
	// Initialize text inputs
	inputs := make([]textinput.Model, 2)

	// Title input
	inputs[formFieldFileName] = textinput.New()
	inputs[formFieldFileName].Placeholder = "Enter file name"
	inputs[formFieldFileName].Focus()
	inputs[formFieldFileName].Width = 50

	// Description input
	inputs[formFieldDescription] = textinput.New()
	inputs[formFieldDescription].Placeholder = "Enter description"
	inputs[formFieldDescription].Width = 50

	return formModel{
		parent:     parent,
		inputs:     inputs,
		focusIndex: 0,
	}
}

// submitForm validates and submits the form
func (m *formModel) submitForm() tea.Cmd {
	// Validate inputs
	filename := strings.TrimSpace(m.inputs[formFieldFileName].Value())
	if filename == "" {
		m.err = fmt.Errorf("filename is required")
		return nil
	}

	description := strings.TrimSpace(m.inputs[formFieldDescription].Value())

	// Set submitting state
	m.isSubmitting = true

	// Determine mime type from file extension
	mimeType := "application/octet-stream" // Default
	ext := filepath.Ext(m.selectedFile)
	if ext != "" {
		if mt := mime.TypeByExtension(ext); mt != "" {
			mimeType = mt
		}
	}

	// Return command to handle the actual submission
	return func() tea.Msg {
		enc, err := enclosure.NewEnclosure(
			m.parent.Ipfs,
			m.selectedFile,
			filename,
			mimeType,
			description,
		)

		if err != nil {
			// Return error message
			return formErrorMsg{err: err}
		}

		var cast *pb.MessageData
		if cast, err = enc.Post(m.parent.Hub, m.parent.Fid, m.parent.PrivateKey); err != nil {
			return formErrorMsg{err: err}
		}

		// Return success message with the created enclosure
		return formSuccessMsg{cast: cast}
	}
}

// formSuccessMsg is sent when an enclosure is successfully created
type formSuccessMsg struct {
	cast *pb.MessageData
}

// formErrorMsg is sent when there's an error creating an enclosure
type formErrorMsg struct {
	err error
}

// Update handles messages and user input for the form
func (m formModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Skip key handling if submitting
		if m.isSubmitting {
			break
		}

		switch msg.String() {
		case "tab", "shift+tab", "up", "down":
			// Cycle focus between inputs
			s := msg.String()

			if s == "up" || s == "shift+tab" {
				m.focusIndex--
				if m.focusIndex < 0 {
					m.focusIndex = len(m.inputs) - 1
				}
			} else {
				m.focusIndex++
				if m.focusIndex >= len(m.inputs) {
					m.focusIndex = 0
				}
			}

			// Focus the selected input
			for i := 0; i < len(m.inputs); i++ {
				if i == m.focusIndex {
					cmds = append(cmds, m.inputs[i].Focus())
				} else {
					m.inputs[i].Blur()
				}
			}

			return m, tea.Batch(cmds...)

		case "enter":
			// Check if on the last input
			if m.focusIndex == len(m.inputs)-1 {
				// Submit the form
				return m, m.submitForm()
			}

			// Otherwise move to next input
			m.focusIndex++
			if m.focusIndex >= len(m.inputs) {
				m.focusIndex = 0
			}

			// Focus the selected input
			for i := 0; i < len(m.inputs); i++ {
				if i == m.focusIndex {
					cmds = append(cmds, m.inputs[i].Focus())
				} else {
					m.inputs[i].Blur()
				}
			}

			return m, tea.Batch(cmds...)

		case "esc":
			// Cancel and go back to list
			m.parent.ShowList()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case formSuccessMsg:
		// Add the new enclosure to the parent model
		// m.parent.AddCast(msg.cast.GetCastAddBody())
		// Go back to list view
		m.parent.ShowList()

	case formErrorMsg:
		// Set error and stop submitting
		m.err = msg.err
		m.isSubmitting = false
	}

	// Handle input updates
	var cmd tea.Cmd
	for i := range m.inputs {
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the form
func (m formModel) View() string {
	var s strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFDF5")).
		Background(lipgloss.Color("#25A065")).
		Padding(0, 1).
		Render("NEW ENCLOSURE")

	s.WriteString(title)
	s.WriteString("\n\n")

	// Selected file
	s.WriteString(fmt.Sprintf("Selected file: %s\n\n", m.selectedFile))

	// Error, if any
	if m.err != nil {
		errText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Render(fmt.Sprintf("Error: %v", m.err))
		s.WriteString(errText)
		s.WriteString("\n\n")
	}

	// Loading state
	if m.isSubmitting {
		s.WriteString("Creating enclosure, please wait...\n\n")
		return s.String()
	}

	// Form fields
	s.WriteString("File Name for downloads:\n")
	s.WriteString(m.inputs[formFieldFileName].View())
	s.WriteString("\n\n")

	s.WriteString("Description:\n")
	s.WriteString(m.inputs[formFieldDescription].View())
	s.WriteString("\n\n")

	// Help text
	s.WriteString("Press Enter to submit, Esc to cancel\n")

	return s.String()
}

// Init implements the tea.Model interface
func (m formModel) Init() tea.Cmd {
	return nil
}
