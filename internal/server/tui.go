package server

import (
	"fmt"
	"strings"
	"time"

	"goMESA/internal/common"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Define the Dracula theme colors
var (
	draculaBackground = lipgloss.Color("#282a36")
	draculaForeground = lipgloss.Color("#f8f8f2")
	draculaSelection  = lipgloss.Color("#44475a")
	draculaComment    = lipgloss.Color("#6272a4")
	draculaPurple     = lipgloss.Color("#bd93f9")
	draculaGreen      = lipgloss.Color("#50fa7b")
	draculaOrange     = lipgloss.Color("#ffb86c")
	draculaPink       = lipgloss.Color("#ff79c6")
	draculaRed        = lipgloss.Color("#ff5555")
	draculaYellow     = lipgloss.Color("#f1fa8c")
	draculaCyan       = lipgloss.Color("#8be9fd")
)

// Define the styles
var (
	// Base styles
	baseStyle = lipgloss.NewStyle().
		Background(draculaBackground).
		Foreground(draculaForeground)

	// Text styles
	titleStyle = lipgloss.NewStyle().
		Background(draculaBackground).
		Foreground(draculaPurple).
		Bold(true).
		MarginLeft(2)

	subtitleStyle = lipgloss.NewStyle().
		Background(draculaBackground).
		Foreground(draculaPink).
		MarginLeft(2)

	infoStyle = lipgloss.NewStyle().
		Background(draculaBackground).
		Foreground(draculaCyan)

	warningStyle = lipgloss.NewStyle().
		Background(draculaBackground).
		Foreground(draculaYellow)

	errorStyle = lipgloss.NewStyle().
		Background(draculaBackground).
		Foreground(draculaRed)

	successStyle = lipgloss.NewStyle().
		Background(draculaBackground).
		Foreground(draculaGreen)

	helpStyle = lipgloss.NewStyle().
		Background(draculaBackground).
		Foreground(draculaComment)

	// Container styles
	windowStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(draculaPurple).
		Padding(1).
		Background(draculaBackground)

	tableStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(draculaCyan).
		Background(draculaBackground)

	tableHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Background(draculaSelection).
		Foreground(draculaPink).
		Align(lipgloss.Center)

	selectedRowStyle = lipgloss.NewStyle().
		Background(draculaSelection).
		Foreground(draculaForeground)

	cellStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Background(draculaBackground).
		Foreground(draculaForeground)

	// Status indicators
	aliveStatusStyle = lipgloss.NewStyle().
		Foreground(draculaGreen)

	miaStatusStyle = lipgloss.NewStyle().
		Foreground(draculaYellow)

	killedStatusStyle = lipgloss.NewStyle().
		Foreground(draculaRed)

	// Input styles
	inputStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(draculaPurple).
		Padding(0, 1).
		Background(draculaBackground).
		Foreground(draculaForeground)

	// Command prompt styles
	promptStyle = lipgloss.NewStyle().
		Background(draculaBackground).
		Foreground(draculaPink).
		Bold(true)

	cursorStyle = lipgloss.NewStyle().
		Background(draculaForeground).
		Foreground(draculaBackground)
)

// Custom key mappings
type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Help     key.Binding
	Quit     key.Binding
	Enter    key.Binding
	Back     key.Binding
	Execute  key.Binding
	Agents   key.Binding
	Interact key.Binding
	DB       key.Binding
	Commands key.Binding
	Ping     key.Binding
	Kill     key.Binding
	Shutdown key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Back, k.Execute},
		{k.Agents, k.Interact, k.DB, k.Commands},
		{k.Ping, k.Kill},
		{k.Help, k.Quit, k.Shutdown},
	}
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "move left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "move right"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "back"),
	),
	Execute: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", "execute"),
	),
	Agents: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "agents"),
	),
	Interact: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "interact"),
	),
	DB: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "database"),
	),
	Commands: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "commands"),
	),
	Ping: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "ping"),
	),
	Kill: key.NewBinding(
		key.WithKeys("k"),
		key.WithHelp("k", "kill"),
	),
	Shutdown: key.NewBinding(
		key.WithKeys("ctrl+q"),
		key.WithHelp("ctrl+q", "shutdown"),
	),
}

// Model types
type sessionState int

const (
	stateMain sessionState = iota
	stateDatabase
	stateInteract
	stateCommand
	stateAgentList
	stateConfirm
	stateCommandOutput
	stateHelp
)

// Item for list
type item struct {
	title       string
	description string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return i.title }

// TUI represents the terminal user interface
type TUI struct {
	db        *Database
	ntpServer *NTPServer
	model     *model
}

// Model holds the state of the TUI
type model struct {
	state           sessionState
	previousState   sessionState
	keys            keyMap
	help            help.Model
	showHelp        bool
	agentTable      table.Model
	agents          []common.Agent
	spinner         spinner.Model
	textInput       textinput.Model
	interactType    string
	interactID      string
	viewport        viewport.Model
	commandHistory  []string
	commandBuffer   string
	confirmMessage  string
	confirmAction   func() error
	statusMessage   string
	statusStyle     lipgloss.Style
	outputContent   string
	outputTitle     string
	width           int
	height          int
	ready           bool
	loading         bool
	db              *Database
	ntpServer       *NTPServer
	mainOptions     list.Model
	dbOptions       list.Model
	interactOptions list.Model
}

// NewTUI creates a new TUI
func NewTUI(db *Database, ntpServer *NTPServer) *TUI {
	return &TUI{
		db:        db,
		ntpServer: ntpServer,
	}
}

// Start starts the TUI
func (t *TUI) Start() error {
	// Initialize the model
	m := initialModel(t.db, t.ntpServer)

	// Create the program
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Run the program
	_, err := p.Run()
	return err
}

// Initialize the model
func initialModel(db *Database, ntpServer *NTPServer) *model {
	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(draculaPurple)

	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = "Enter command..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 80
	ti.Prompt = promptStyle.Render("MESA ~ ")
	ti.TextStyle = lipgloss.NewStyle().Foreground(draculaForeground)
	ti.Cursor.Style = cursorStyle

	// Initialize help
	h := help.New()
	h.Style = helpStyle

	// Initialize viewport for output
	vp := viewport.New(80, 20)
	vp.Style = windowStyle

	// Initialize agent table
	columns := []table.Column{
		{Title: "Agent IP", Width: 15},
		{Title: "OS", Width: 10},
		{Title: "Service", Width: 15},
		{Title: "Status", Width: 10},
		{Title: "Last Seen", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	t.SetStyles(table.Styles{
		Header:   tableHeaderStyle,
		Selected: selectedRowStyle,
		Cell:     cellStyle,
	})

	// Initialize main menu options
	mainItems := []list.Item{
		item{title: "🖥️  Agents", description: "Display the board of agent entries"},
		item{title: "💾  Database", description: "Enter the database subprompt"},
		item{title: "🔄  Interact", description: "Enter the interact subprompt"},
		item{title: "❓  Help", description: "Display help information"},
		item{title: "🚪  Exit", description: "Quit the program, state will be saved"},
		item{title: "⚠️  Shutdown", description: "Quit the program, all agents are killed, database is cleaned"},
	}

	mainMenu := list.New(mainItems, list.NewDefaultDelegate(), 0, 0)
	mainMenu.Title = "MESA C2 - Main Menu"
	mainMenu.SetShowStatusBar(false)
	mainMenu.SetFilteringEnabled(false)
	mainMenu.SetShowHelp(false)
	mainMenu.Styles.Title = titleStyle

	// Initialize database menu options
	dbItems := []list.Item{
		item{title: "🖥️  Agents", description: "Display the board of agent entries"},
		item{title: "🏷️  Group", description: "Add a service identifier to an agent"},
		item{title: "🗑️  Remove All", description: "Remove all agents from the database"},
		item{title: "📊  Meta", description: "Describe the agent tables metadata"},
		item{title: "❓  Help", description: "Display help information"},
		item{title: "⬅️  Back", description: "Return to the main prompt"},
	}

	dbMenu := list.New(dbItems, list.NewDefaultDelegate(), 0, 0)
	dbMenu.Title = "MESA C2 - Database Menu"
	dbMenu.SetShowStatusBar(false)
	dbMenu.SetFilteringEnabled(false)
	dbMenu.SetShowHelp(false)
	dbMenu.Styles.Title = titleStyle

	// Initialize interact menu options
	interactItems := []list.Item{
		item{title: "🖥️  Agents", description: "Display agents under the interact filters"},
		item{title: "📡  Ping", description: "Ping agent"},
		item{title: "⚠️  Kill", description: "Send kill command to agent"},
		item{title: "💻  Command", description: "Enter the cmd subprompt"},
		item{title: "❓  Help", description: "Display help information"},
		item{title: "⬅️  Back", description: "Return to the main prompt"},
	}

	interactMenu := list.New(interactItems, list.NewDefaultDelegate(), 0, 0)
	interactMenu.Title = "MESA C2 - Interact Menu"
	interactMenu.SetShowStatusBar(false)
	interactMenu.SetFilteringEnabled(false)
	interactMenu.SetShowHelp(false)
	interactMenu.Styles.Title = titleStyle

	return &model{
		state:           stateMain,
		previousState:   stateMain,
		keys:            keys,
		help:            h,
		spinner:         s,
		textInput:       ti,
		viewport:        vp,
		statusStyle:     infoStyle,
		db:              db,
		ntpServer:       ntpServer,
		mainOptions:     mainMenu,
		dbOptions:       dbMenu,
		interactOptions: interactMenu,
		agentTable:      t,
	}
}

// Init initializes the model
func (m *model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.updateAgentList,
	)
}

// Update handles events and updates the model
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			if m.state != stateConfirm {
				return m, tea.Quit
			}

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp

		case key.Matches(msg, m.keys.Back):
			return m.handleBackKey()

		case key.Matches(msg, m.keys.Agents):
			if m.state == stateMain || m.state == stateDatabase || m.state == stateInteract {
				return m, m.updateAgentList
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Update the viewport
		headerHeight := 6
		footerHeight := 3
		verticalMarginHeight := headerHeight + footerHeight

		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - verticalMarginHeight

		// Update the table
		m.agentTable.SetWidth(msg.Width - 4)
		m.agentTable.SetHeight(msg.Height - verticalMarginHeight)

		// Update the text input
		m.textInput.Width = msg.Width - 20

		// Update the menus
		m.mainOptions.SetSize(msg.Width-4, msg.Height-verticalMarginHeight)
		m.dbOptions.SetSize(msg.Width-4, msg.Height-verticalMarginHeight)
		m.interactOptions.SetSize(msg.Width-4, msg.Height-verticalMarginHeight)

	case agentListMsg:
		m.agents = msg
		m.loading = false

		// Update the table
		var rows []table.Row
		for _, agent := range m.agents {
			// Format status with appropriate style and emoji
			var statusStr string
			switch agent.Status {
			case common.AgentStatusAlive:
				statusStr = aliveStatusStyle.Render("✅ ALIVE")
			case common.AgentStatusMissing:
				statusStr = miaStatusStyle.Render("⚠️ MIA")
			case common.AgentStatusKilled:
				statusStr = killedStatusStyle.Render("❌ KILLED")
			default:
				statusStr = agent.Status
			}

			rows = append(rows, table.Row{
				agent.ID,
				agent.OS,
				agent.Service,
				statusStr,
				agent.LastSeen.Format("2006-01-02 15:04:05"),
			})
		}

		m.agentTable.SetRows(rows)
		return m, nil

	case statusMsg:
		m.statusMessage = msg.message
		m.statusStyle = msg.style
		m.loading = false

		// Clear status after a delay
		return m, tea.Sequence(
			tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
				return statusMsg{message: "", style: infoStyle}
			}),
		)

	case commandOutputMsg:
		m.outputContent = msg.output
		m.outputTitle = msg.title
		m.state = stateCommandOutput
		m.viewport.SetContent(m.outputContent)
		m.loading = false
		return m, nil

	case spinner.TickMsg:
		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		return m, spinnerCmd
	}

	// Handle state-specific updates
	switch m.state {
	case stateMain:
		m.mainOptions, cmd = m.mainOptions.Update(msg)
		cmds = append(cmds, cmd)

		if _, ok := msg.(tea.KeyMsg); ok {
			// Handle list selection
			if m.mainOptions.SelectedItem() != nil {
				selectedItem := m.mainOptions.SelectedItem().(item)
				if key.Matches(msg, m.keys.Enter) {
					switch selectedItem.title {
					case "🖥️  Agents":
						m.loading = true
						return m, m.updateAgentList
					case "💾  Database":
						m.state = stateDatabase
						return m, nil
					case "🔄  Interact":
						// Show agent list for interaction
						m.loading = true
						m.previousState = stateMain
						m.state = stateAgentList
						return m, m.updateAgentList
					case "❓  Help":
						m.showHelp = !m.showHelp
						return m, nil
					case "🚪  Exit":
						return m, tea.Quit
					case "⚠️  Shutdown":
						m.confirmMessage = "Are you sure you want to shutdown? This will kill all agents and clean the database."
						m.confirmAction = func() error {
							// Kill all agents
							agents, err := m.db.GetAllAgents()
							if err == nil {
								for _, agent := range agents {
									m.ntpServer.SendKillCommand(agent.ID)
								}
							}

							// Clean the database
							err = m.db.CleanDatabase()
							if err != nil {
								return err
							}

							return nil
						}
						m.previousState = stateMain
						m.state = stateConfirm
						return m, nil
					}
				}
			}
		}

	case stateDatabase:
		m.dbOptions, cmd = m.dbOptions.Update(msg)
		cmds = append(cmds, cmd)

		if _, ok := msg.(tea.KeyMsg); ok {
			// Handle list selection
			if m.dbOptions.SelectedItem() != nil {
				selectedItem := m.dbOptions.SelectedItem().(item)
				if key.Matches(msg, m.keys.Enter) {
					switch selectedItem.title {
					case "🖥️  Agents":
						m.loading = true
						return m, m.updateAgentList
					case "🏷️  Group":
						// Handle group command input
						m.textInput.Placeholder = "Enter: <ip> <os/service> <name>"
						m.textInput.Focus()
						m.previousState = stateDatabase
						// Custom handler for the group command
						return m, func() tea.Msg {
							// This would normally process the group command
							return nil
						}
					case "🗑️  Remove All":
						m.confirmMessage = "Are you sure you want to remove all agents from the database?"
						m.confirmAction = func() error {
							return m.db.RemoveAllAgents()
						}
						m.previousState = stateDatabase
						m.state = stateConfirm
						return m, nil
					case "📊  Meta":
						// Display database metadata
						content := "Agents Table Schema:\n"
						content += "- agent_id: VARCHAR(45) PRIMARY KEY\n"
						content += "- os: VARCHAR(255)\n"
						content += "- service: VARCHAR(255)\n"
						content += "- status: VARCHAR(20) DEFAULT 'ALIVE'\n"
						content += "- first_seen: TIMESTAMP DEFAULT CURRENT_TIMESTAMP\n"
						content += "- last_seen: TIMESTAMP DEFAULT CURRENT_TIMESTAMP\n"
						content += "- network_adapter: VARCHAR(255)\n\n"

						content += "Commands Table Schema:\n"
						content += "- id: INT AUTO_INCREMENT PRIMARY KEY\n"
						content += "- agent_id: VARCHAR(45) FOREIGN KEY REFERENCES agents(agent_id)\n"
						content += "- content: TEXT\n"
						content += "- timestamp: TIMESTAMP DEFAULT CURRENT_TIMESTAMP\n"
						content += "- status: VARCHAR(20) DEFAULT 'SENT'\n"
						content += "- output: TEXT\n"

						m.outputContent = content
						m.outputTitle = "Database Metadata"
						m.state = stateCommandOutput
						m.viewport.SetContent(content)
						return m, nil
					case "❓  Help":
						m.showHelp = !m.showHelp
						return m, nil
					case "⬅️  Back":
						m.state = stateMain
						return m, nil
					}
				}
			}
		}

	case stateInteract:
		m.interactOptions, cmd = m.interactOptions.Update(msg)
		cmds = append(cmds, cmd)

		if _, ok := msg.(tea.KeyMsg); ok {
			// Handle list selection
			if m.interactOptions.SelectedItem() != nil {
				selectedItem := m.interactOptions.SelectedItem().(item)
				if key.Matches(msg, m.keys.Enter) {
					switch selectedItem.title {
					case "🖥️  Agents":
						// Show filtered agent list
						m.loading = true
						return m, m.updateFilteredAgentList
					case "📡  Ping":
						// Send ping command
						m.loading = true
						return m, func() tea.Msg {
							err := m.sendPingCommand()
							if err != nil {
								return statusMsg{
									message: fmt.Sprintf("Error sending ping: %v", err),
									style:   errorStyle,
								}
							}
							return statusMsg{
								message: fmt.Sprintf("Ping sent to %s (%s)", m.interactID, m.interactType),
								style:   successStyle,
							}
						}
					case "⚠️  Kill":
						// Send kill command with confirmation
						m.confirmMessage = fmt.Sprintf("Are you sure you want to kill %s (%s)?", m.interactID, m.interactType)
						m.confirmAction = func() error {
							return m.sendKillCommand()
						}
						m.previousState = stateInteract
						m.state = stateConfirm
						return m, nil
					case "💻  Command":
						// Enter command mode
						m.state = stateCommand
						m.textInput.Placeholder = "Enter command to execute..."
						m.textInput.Focus()
						// Update prompt
						m.textInput.Prompt = promptStyle.Render(fmt.Sprintf("MESA {%s/%s/CMD} ~ ", m.interactType, m.interactID))
						return m, nil
					case "❓  Help":
						m.showHelp = !m.showHelp
						return m, nil
					case "⬅️  Back":
						m.state = stateMain
						return m, nil
					}
				}
			}
		}

	case stateCommand:
		// Handle command input
		var textInputCmd tea.Cmd
		m.textInput, textInputCmd = m.textInput.Update(msg)
		cmds = append(cmds, textInputCmd)

		// Process command on enter
		if _, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(msg, m.keys.Enter) {
				command := m.textInput.Value()
				if command == "back" {
					m.state = stateInteract
					return m, nil
				} else if command == "help" {
					m.showHelp = !m.showHelp
					return m, nil
				} else if command != "" {
					// Execute the command
					m.loading = true
					m.textInput.Reset()

					return m, func() tea.Msg {
						output, err := m.executeCommand(command)
						if err != nil {
							return statusMsg{
								message: fmt.Sprintf("Error executing command: %v", err),
								style:   errorStyle,
							}
						}

						return commandOutputMsg{
							output: output,
							title:  fmt.Sprintf("Command Output: %s", command),
						}
					}
				}
			}
		}

	case stateAgentList:
		var tableCmd tea.Cmd
		m.agentTable, tableCmd = m.agentTable.Update(msg)
		cmds = append(cmds, tableCmd)

		if _, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(msg, m.keys.Enter) {
				// Select agent for interaction
				if len(m.agentTable.Rows()) > 0 {
					selectedRow := m.agentTable.SelectedRow()
					if len(selectedRow) >= 1 {
						m.interactID = selectedRow[0]
						m.interactType = "agent_id"
						m.state = stateInteract

						// Update interact menu title
						m.interactOptions.Title = fmt.Sprintf("MESA C2 - Interact with %s", m.interactID)

						// Update text input prompt
						m.textInput.Prompt = promptStyle.Render(fmt.Sprintf("MESA {%s/%s} ~ ", m.interactType, m.interactID))

						return m, statusMsg{
							message: fmt.Sprintf("Now interacting with agent %s", m.interactID),
							style:   successStyle,
						}
					}
				}
			}
		}

	case stateConfirm:
		if _, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "y", "Y":
				// Execute the confirm action
				m.loading = true
				return m, func() tea.Msg {
					err := m.confirmAction()
					if err != nil {
						return statusMsg{
							message: fmt.Sprintf("Error: %v", err),
							style:   errorStyle,
						}
					}

					if m.confirmMessage == "Are you sure you want to shutdown? This will kill all agents and clean the database." {
						// If it was a shutdown confirm, quit the application
						return tea.Quit()
					}

					m.state = m.previousState
					return statusMsg{
						message: "Action completed successfully",
						style:   successStyle,
					}
				}
			case "n", "N":
				// Cancel the confirm action
				m.state = m.previousState
				return m, statusMsg{
					message: "Action cancelled",
					style:   infoStyle,
				}
			}
		}

	case stateCommandOutput:
		// Handle scrolling in the viewport
		var viewportCmd tea.Cmd
		m.viewport, viewportCmd = m.viewport.Update(msg)
		cmds = append(cmds, viewportCmd)

		if _, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(msg, m.keys.Back) || key.Matches(msg, m.keys.Enter) {
				m.state = m.previousState
				return m, nil
			}
		}
	}

	// Return the updated model and commands
	return m, tea.Batch(cmds...)
}

// Handle back key based on current state
func (m *model) handleBackKey() (tea.Model, tea.Cmd) {
	switch m.state {
	case stateMain:
		// Do nothing, already at main

	case stateDatabase:
		m.state = stateMain

	case stateInteract:
		m.state = stateMain

	case stateCommand:
		m.state = stateInteract
		m.textInput.Prompt = promptStyle.Render(fmt.Sprintf("MESA {%s/%s} ~ ", m.interactType, m.interactID))

	case stateAgentList:
		m.state = m.previousState

	case stateConfirm:
		m.state = m.previousState

	case stateCommandOutput:
		m.state = m.previousState
	}

	return m, nil
}

// View renders the UI
func (m *model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var content string

	// Header
	header := titleStyle.Render("🔮 MESA C2 Framework") + "\n"

	// Different view based on state
	switch m.state {
	case stateMain:
		content = m.mainOptions.View()

	case stateDatabase:
		content = m.dbOptions.View()

	case stateInteract:
		content = m.interactOptions.View()

	case stateCommand:
		content = "\n  " + successStyle.Render(fmt.Sprintf("Executing commands on %s (%s)", m.interactID, m.interactType)) + "\n\n"
		content += "  " + m.textInput.View() + "\n\n"

		if len(m.commandHistory) > 0 {
			content += subtitleStyle.Render("  Command History:") + "\n"
			for i := len(m.commandHistory) - 1; i >= 0 && i >= len(m.commandHistory)-5; i-- {
				content += fmt.Sprintf("  %s\n", m.commandHistory[i])
			}
		}

	case stateAgentList:
		content = titleStyle.Render("  Select an Agent") + "\n\n"
		content += m.agentTable.View()

	case stateConfirm:
		content = errorStyle.Render("  ⚠️  "+m.confirmMessage) + "\n\n"
		content += "  Press 'y' to confirm or 'n' to cancel.\n"

	case stateCommandOutput:
		content = titleStyle.Render("  "+m.outputTitle) + "\n\n"
		content += m.viewport.View()
	}

	// Status message
	var status string
	if m.loading {
		status = m.spinner.View() + " Loading..."
	} else if m.statusMessage != "" {
		status = m.statusStyle.Render(m.statusMessage)
	}

	// Footer with help
	footer := "\n"
	if m.showHelp {
		footer += m.help.View(m.keys)
	} else {
		footer += helpStyle.Render("Press ? for help, q to quit")
	}

	// Put it all together
	return windowStyle.Render(fmt.Sprintf("%s\n%s\n%s%s", header, content, status, footer))
}

// Custom messages
type agentListMsg []common.Agent

type statusMsg struct {
	message string
	style   lipgloss.Style
}

type commandOutputMsg struct {
	output string
	title  string
}

// Model methods
func (m *model) updateAgentList() tea.Msg {
	agents, err := m.db.GetAllAgents()
	if err != nil {
		return statusMsg{
			message: fmt.Sprintf("Error fetching agents: %v", err),
			style:   errorStyle,
		}
	}

	return agentListMsg(agents)
}

func (m *model) updateFilteredAgentList() tea.Msg {
	agents, err := m.db.GetAgentsByGroup(m.interactType, m.interactID)
	if err != nil {
		return statusMsg{
			message: fmt.Sprintf("Error fetching agents: %v", err),
			style:   errorStyle,
		}
	}

	return agentListMsg(agents)
}

func (m *model) sendPingCommand() error {
	agents, err := m.db.GetAgentsByGroup(m.interactType, m.interactID)
	if err != nil {
		return err
	}

	for _, agent := range agents {
		err := m.ntpServer.SendPingCommand(agent.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *model) sendKillCommand() error {
	agents, err := m.db.GetAgentsByGroup(m.interactType, m.interactID)
	if err != nil {
		return err
	}

	for _, agent := range agents {
		err := m.ntpServer.SendKillCommand(agent.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *model) executeCommand(command string) (string, error) {
	// Add command to history
	m.commandHistory = append(m.commandHistory, command)

	// Execute the command on all matching agents
	agents, err := m.db.GetAgentsByGroup(m.interactType, m.interactID)
	if err != nil {
		return "", err
	}

	var output strings.Builder
	for _, agent := range agents {
		output.WriteString(fmt.Sprintf("=== Output from %s ===\n", agent.ID))
		result, err := m.ntpServer.ExecuteCommand(agent.ID, command)
		if err != nil {
			output.WriteString(fmt.Sprintf("Error: %v\n\n", err))
		} else {
			output.WriteString(fmt.Sprintf("%s\n\n", result))
		}
	}

	return output.String(), nil
}
