package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// fileEntry represents a file or directory in the file picker
type fileEntry struct {
	name  string
	path  string
	isDir bool
}

// filePickerModel displays a file picker
type filePickerModel struct {
	parent       *MainModel
	currentDir   string
	files        []fileEntry
	cursor       int
	scrollOffset int
	width        int
	height       int
	maxVisible   int
	err          error
}

// newFilePickerModel creates a new file picker model
func newFilePickerModel(parent *MainModel, startDir string) filePickerModel {
	// Use a more reasonable default for terminal height
	// Most terminals are at least 24 rows tall
	maxVisible := 50

	m := filePickerModel{
		parent:     parent,
		currentDir: startDir,
		maxVisible: maxVisible,
	}

	// Load initial files
	m.loadFiles()

	return m
}

// loadFiles loads the files in the current directory
func (m *filePickerModel) loadFiles() {
	entries, err := os.ReadDir(m.currentDir)
	if err != nil {
		m.err = err
		return
	}

	m.files = make([]fileEntry, 0, len(entries))

	// Add parent directory entry if not at root
	if filepath.Dir(m.currentDir) != m.currentDir {
		m.files = append(m.files, fileEntry{
			name:  "..",
			path:  filepath.Dir(m.currentDir),
			isDir: true,
		})
	}

	// Add all directories and files
	for _, entry := range entries {
		// Skip hidden files
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		fullPath := filepath.Join(m.currentDir, entry.Name())

		m.files = append(m.files, fileEntry{
			name:  entry.Name(),
			path:  fullPath,
			isDir: entry.IsDir(),
		})
	}

	// Sort: directories first, then files, both alphabetically
	sort.Slice(m.files, func(i, j int) bool {
		if m.files[i].isDir && !m.files[j].isDir {
			return true
		}
		if !m.files[i].isDir && m.files[j].isDir {
			return false
		}
		return m.files[i].name < m.files[j].name
	})

	// Reset cursor and scroll
	m.cursor = 0
	m.scrollOffset = 0
}

// Update handles messages and user input for the file picker
func (m filePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			// Move cursor down
			if m.cursor < len(m.files)-1 {
				m.cursor++
				// Adjust scroll if needed
				if m.cursor-m.scrollOffset >= m.maxVisible {
					m.scrollOffset++
				}
			}
		case "k", "up":
			// Move cursor up
			if m.cursor > 0 {
				m.cursor--
				// Adjust scroll if needed
				if m.cursor < m.scrollOffset {
					m.scrollOffset--
				}
			}
		case "right":
			// Enter the selected directory (if it's a directory)
			if m.cursor < len(m.files) {
				selected := m.files[m.cursor]
				if selected.isDir {
					// Change directory
					m.currentDir = selected.path
					m.loadFiles()
				}
			}
		case "left":
			// Go up one directory (parent directory)
			if filepath.Dir(m.currentDir) != m.currentDir {
				m.currentDir = filepath.Dir(m.currentDir)
				m.loadFiles()
			}
		case "enter":
			// Handle file/directory selection
			if m.cursor < len(m.files) {
				selected := m.files[m.cursor]
				if selected.isDir {
					// Change directory
					m.currentDir = selected.path
					m.loadFiles()
				} else {
					// Select file
					m.parent.ShowForm(selected.path)
				}
			}
		case "esc":
			// Cancel and go back to list
			m.parent.ShowList()
		}
	case tea.WindowSizeMsg:
		// Store the window dimensions
		m.width = msg.Width
		m.height = msg.Height

		// Use almost all available height for the file list
		// Reserve 6 lines for header, footer and scroll indicators
		m.maxVisible = m.height - 3

		// Set reasonable bounds
		if m.maxVisible < 3 {
			m.maxVisible = 3 // Minimum
		}

		// Print dimensions for debugging
		//fmt.Printf("Window size: %dx%d, maxVisible: %d\n", m.width, m.height, m.maxVisible)
	}

	return m, nil
}

// View renders the file picker
func (m filePickerModel) View() string {
	var s strings.Builder

	// Title with current directory
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFDF5")).
		Background(lipgloss.Color("#2D7D9A")).
		Padding(0, 1).
		Render("SELECT A FILE - " + m.currentDir)

	s.WriteString(title)
	s.WriteString("\n")

	// Error message (if any)
	if m.err != nil {
		errText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Render(fmt.Sprintf("Error: %v", m.err))
		s.WriteString(errText)
		s.WriteString("\n")
	}

	// Empty state or file list
	contentHeight := m.height - 3 // Title + error potential + nav bar + buffer
	if m.err != nil {
		contentHeight--
	}

	if len(m.files) == 0 {
		// Empty directory message
		s.WriteString("No files in this directory.\n")

		// Fill remaining space
		if contentHeight > 1 { // 1 line already used for message
			s.WriteString(strings.Repeat("\n", contentHeight-1))
		}
	} else {
		// Calculate visible range
		end := m.scrollOffset + m.maxVisible
		if end > len(m.files) {
			end = len(m.files)
		}
		visibleFiles := m.files[m.scrollOffset:end]

		// Show scroll indicator if needed (in a more compact way)
		if m.scrollOffset > 0 {
			s.WriteString("↑ \n")
		} else {
			s.WriteString("  \n")
		}
		contentHeight -= 1 // Used for scroll indicator

		// List files
		for i, file := range visibleFiles {
			cursor := " "
			if m.cursor == i+m.scrollOffset {
				cursor = ">"
			} else {
				cursor = " "
			}

			// Format the entry with D for directories, F for files
			prefix := "F"
			if file.isDir {
				prefix = "D"
			}

			line := fmt.Sprintf("%s %s %s", cursor, prefix, file.name)

			if m.cursor == i+m.scrollOffset {
				// Highlight selected item
				line = lipgloss.NewStyle().
					Bold(true).
					Padding(0, 0).
					Render(line)
			}

			s.WriteString(line)
			s.WriteString("\n")
			contentHeight--
		}

		// Show scroll indicator if needed (in a more compact way)
		if end < len(m.files) {
			s.WriteString("↓ \n")
			contentHeight--
		}

		// Fill remaining space if any
		if contentHeight > 0 {
			s.WriteString(strings.Repeat("\n", contentHeight))
		}
	}

	s.WriteString("↑/↓/←/→ navigate • ENTER select • ESC cancel")

	return s.String()
}

// Init implements the tea.Model interface
func (m filePickerModel) Init() tea.Cmd {
	return nil
}
