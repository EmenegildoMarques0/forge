package main

import (
	"fmt"
	"forge/internal/ai"
	"forge/internal/git"
	"forge/internal/ui"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateMenu state = iota
	stateLoading
	stateResult
)

type model struct {
	choices     []string
	cursor      int
	state       state
	message     string
	aiResult    string
	stagedFiles []string // Nova lista para armazenar os arquivos alterados
}

func initialModel() model {
	return model{
		choices: []string{"🤖 Forjar Commit (IA)", "📝 Gerar README", "⚙️  Configurações"},
		state:   stateMenu,
	}
}

func (m model) Init() tea.Cmd { return nil }

type aiResponseMsg string
type errMsg error

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.state != stateMenu {
				m.state = stateMenu
				m.message = ""
				return m, nil
			}
		}

		if m.state == stateMenu {
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.choices)-1 {
					m.cursor++
				}
			case "enter":
				if m.cursor == 0 {
					m.state = stateLoading
					return m, m.generateCommitAction()
				}
			}
		} else if m.state == stateResult {
			if msg.String() == "enter" {
				return m, tea.Quit
			}
		}

	case aiResponseMsg:
		m.state = stateResult
		m.aiResult = string(msg)
		// Busca a lista de arquivos alterados após o processamento
		files, _ := git.GetStagedFiles()
		m.stagedFiles = files
		return m, nil

	case errMsg:
		m.state = stateMenu
		m.message = "❌ " + msg.Error()
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	var body string

	switch m.state {
	case stateLoading:
		body = "\n  " + lipgloss.NewStyle().Foreground(ui.EclipseCyan).Render("⏳ Adicionando arquivos e forjando inteligência...")

	case stateResult:
		body = ui.SelectedItem.Render("💡 Sugestão da IA:") + "\n\n"
		body += lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ui.EclipseCyan).
			Padding(1).
			Width(60).
			Render(m.aiResult)

		// Seção de arquivos alterados
		if len(m.stagedFiles) > 0 {
			body += "\n\n" + lipgloss.NewStyle().Foreground(ui.EclipseBlue).Bold(true).Render("📦 Arquivos Alterados:") + "\n"
			for _, file := range m.stagedFiles {
				body += fmt.Sprintf("  %s %s\n", lipgloss.NewStyle().Foreground(ui.EclipseCyan).Render("•"), file)
			}
		}

		body += "\n" + ui.HelpStyle.Render("[Enter] Confirmar Commit • [Esc] Voltar")

	default: // stateMenu
		body = "O que vamos forjar hoje?\n\n"
		if m.message != "" {
			body += lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Render(m.message) + "\n\n"
		}
		for i, choice := range m.choices {
			cursor := "  "
			if m.cursor == i {
				cursor = lipgloss.NewStyle().Foreground(ui.EclipseCyan).Render("👉")
				body += fmt.Sprintf("%s %s\n", cursor, ui.SelectedItem.Render(choice))
			} else {
				body += fmt.Sprintf("%s %s\n", cursor, choice)
			}
		}
		body += "\n" + ui.HelpStyle.Render("pressione ↑/↓ para navegar • enter para selecionar • q para sair")
	}

	header := ui.Header.Render(" 🌑 FORGE | Eclipse Dynamics ") + "\n\n"
	return ui.MainBox.Render(header+body) + "\n"
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Houve um erro: %v", err)
		os.Exit(1)
	}
}

func (m model) generateCommitAction() tea.Cmd {
	return func() tea.Msg {
		git.AddAll()
		diff, err := git.GetDiff()
		if err != nil {
			return errMsg(err)
		}

		// AGORA CHAMAMOS A IA REAL
		message, err := ai.GenerateCommitMessage(diff)
		if err != nil {
			return errMsg(err)
		}

		return aiResponseMsg(message)
	}
}
