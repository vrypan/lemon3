package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// listModel displays a list of enclosures
type listModel struct {
	parent    *MainModel
	cursor    int
	width     int
	height    int
	viewStart int
	viewEnd   int
}

// newListModel creates a new list model
func newListModel(parent *MainModel) listModel {
	return listModel{
		parent: parent,
	}
}

// Update handles messages and user input for the list view
func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.moveCursorDown()
		case "k", "up":
			m.moveCursorUp()
		case "u":
			if m.parent.PrivateKey != nil {
				m.parent.ShowFilePicker()
			} else {
				m.parent.statusMsg = "No private key configured. Use `lemon3 config` to configure a private key."
			}

			return m, nil
		case "enter", "right":
			m.download()

		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// View renders the list view
func (m listModel) View() string {
	var s strings.Builder
	usedLines := 0
	// Title
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFDF5")).
		Background(lipgloss.Color("#25A065")).
		Padding(0, 0).
		Render(" -- Lemon3 -- ")

	s.WriteString(title)
	s.WriteString("\n\n")
	usedLines += 2

	// Empty state

	if m.viewEnd == 0 {
		m.initViewport()
	}
	height := 0
	if m.parent.casts != nil {
		for i := m.viewStart; i <= m.viewEnd; i++ {
			line := m.parent.casts[i].Rendered
			if m.cursor == i {
				s.WriteString(lipgloss.NewStyle().
					Bold(true).
					Render(line))
			} else {
				s.WriteString(lipgloss.NewStyle().
					Padding(0, 0).
					Render(line))
			}
			s.WriteString("\n")
			height += m.parent.casts[i].Height + 1
		}

	}

	// Fill space to push navigation bar to bottom if needed
	remainingLines := m.height - height - 4 // -1 for navigation bar
	if remainingLines > 0 {
		s.WriteString(strings.Repeat("\n", remainingLines))
	}
	// Add navigation bar at the bottom
	s.WriteString("↑/↓ navigate")
	s.WriteString(" • u upload")
	s.WriteString(" • ENTER download")
	s.WriteString(" • q quit")
	s.WriteString("\n")
	if m.parent.statusMsg != "" {
		s.WriteString(
			lipgloss.NewStyle().
				Padding(0, 0).
				Bold(true).
				Render(m.parent.statusMsg))
	}
	return s.String()
}

// Init implements the tea.Model interface
func (m listModel) Init() tea.Cmd {
	return nil
}

func (m *listModel) initViewport() {
	m.viewStart = 0
	height := 0
	for i, c := range m.parent.casts {
		if height+c.Height+1 > m.WindowHeight() {
			m.viewEnd = i - 1
			m.cursor = 0
			break
		}
		height += c.Height + 1
		m.viewEnd = i
	}
}

func (m *listModel) moveCursorUp() {
	if m.cursor > 0 {
		m.cursor--
		if m.cursor < m.viewStart {
			m.viewStart = m.cursor
			m.recalculateViewEnd()
		}
	}
}

func (m *listModel) moveCursorDown() {
	if m.viewEnd == 0 {
		m.initViewport()
	}
	if m.cursor < len(m.parent.casts)-1 {
		m.cursor++
		if m.cursor > m.viewEnd {
			m.viewEnd = m.cursor
			m.recalculateViewStart()
		}
	}
}

func (m *listModel) recalculateViewEnd() {
	height := 0
	for i := m.viewStart; i < len(m.parent.casts); i++ {
		if height+m.parent.casts[i].Height+1 > m.WindowHeight() {
			m.viewEnd = i - 1
			return
		}
		height += m.parent.casts[i].Height + 1
	}
	m.viewEnd = len(m.parent.casts)
}

func (m *listModel) recalculateViewStart() {
	height := 0
	i := m.viewEnd
	for {
		height += m.parent.casts[i].Height + 1
		if height > m.WindowHeight() {
			m.viewStart = i + 1
			return
		}
		i--
		if i < 0 {
			m.viewStart = 0
			return
		}
	}
}

func (m listModel) WindowHeight() int {
	return m.height - 4
}

func (m listModel) download() {
	if m.parent.casts[m.cursor].Enclosure == nil {
		m.parent.statusMsg = "Fetching enclosure..."
		err := m.parent.casts[m.cursor].GetEnclosure(m.parent.Ipfs)
		if err != nil {
			m.parent.statusMsg = "Error fetching enclosure"
			return
		}
		m.parent.casts[m.cursor].Render(m.parent.Hub)
		m.parent.statusMsg = ""
		return
	}

	e := m.parent.casts[m.cursor].Enclosure
	m.parent.statusMsg = fmt.Sprintf("Downloading %s...", e.FileName)
	err := m.parent.Ipfs.GetFile(e.FileCID, "./"+e.FileName)
	if err != nil {
		m.parent.statusMsg = fmt.Sprintf("Error downloading file: %s", e.FileName)
		return
	}
	m.parent.statusMsg = fmt.Sprintf("Downloaded: %s", e.FileName)
}
