package main

import (
	"fmt"
	"forge/internal/ai"
	"forge/internal/config"
	"forge/internal/git"
	"forge/internal/ui"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput" // Import necessário para capturar a chave
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateMenu state = iota
	stateLoading
	stateResult
	statePushing
	stateConfig // Estado adicionado para gerenciar a tela de configurações
)

type model struct {
	choices     []string
	cursor      int
	state       state
	message     string
	aiResult    string
	stagedFiles []string
	textInput   textinput.Model // Componente para manipulação de digitação
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Cole aqui a sua GEMINI_API_KEY (AIzaSy...)"
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50

	// Tenta carregar a chave existente para já iniciar o input preenchido
	if cfg, err := config.Load(); err == nil && cfg.GeminiAPIKey != "" {
		ti.SetValue(cfg.GeminiAPIKey)
	}

	return model{
		choices:   []string{"🤖 Forjar Commit (IA)", "📝 Gerar README", "⚙️  Configurações"},
		state:     stateMenu,
		textInput: ti,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink // Faz o cursor do input piscar nativamente
}

type aiResponseMsg string
type commitSuccessMsg string
type errMsg error

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.state == stateMenu {
				return m, tea.Quit
			}
		case "esc":
			if m.state != stateMenu && m.state != statePushing {
				m.state = stateMenu
				m.message = ""
				return m, nil
			}
		}

		// Intercepta e processa as teclas se o usuário estiver na tela de Configuração
		if m.state == stateConfig {
			switch msg.String() {
			case "enter":
				key := strings.TrimSpace(m.textInput.Value())
				if key == "" {
					m.message = "❌ A chave não pode estar vazia!"
					return m, nil
				}
				err := config.Save(key)
				if err != nil {
					m.message = "❌ Erro ao guardar configuração: " + err.Error()
					return m, nil
				}
				m.state = stateMenu
				m.message = "✅ Configuração gravada com sucesso!"
				return m, nil
			}

			// Atualiza o estado interno do input de texto com o caractere digitado
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		// Processa navegação e ações padrão se estiver no Menu Principal
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
				} else if m.cursor == 2 { // Selecionou a opção "⚙️ Configurações"
					m.state = stateConfig
					m.message = ""
					return m, nil
				}
			}
		} else if m.state == stateResult {
			if msg.String() == "enter" {
				m.state = statePushing
				return m, m.executeCommitAndPushAction()
			}
		}

	case aiResponseMsg:
		m.state = stateResult
		m.aiResult = string(msg)
		files, _ := git.GetStagedFiles()
		m.stagedFiles = files
		return m, nil

	case commitSuccessMsg:
		return m, tea.Quit

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

	case statePushing:
		body = "\n  " + lipgloss.NewStyle().Foreground(ui.EclipseCyan).Render("🚀 Gravando commit e empurrando para o GitHub...")

	case stateConfig:
		body = ui.SelectedItem.Render("⚙️  Configuração do Forge") + "\n\n"
		if m.message != "" {
			body += lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Render(m.message) + "\n\n"
		}
		body += "Introduza o seu token do Gemini API:\n\n"
		body += m.textInput.View() + "\n\n"
		body += ui.HelpStyle.Render("[Enter] Guardar Chave • [Esc] Voltar ao Menu")

	case stateResult:
		body = ui.SelectedItem.Render("💡 Sugestão da IA:") + "\n\n"
		body += lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ui.EclipseCyan).
			Padding(1).
			Width(60).
			Render(m.aiResult)

		if len(m.stagedFiles) > 0 {
			body += "\n\n" + lipgloss.NewStyle().Foreground(ui.EclipseBlue).Bold(true).Render("📦 Arquivos Alterados:") + "\n"
			for _, file := range m.stagedFiles {
				body += fmt.Sprintf("  %s %s\n", lipgloss.NewStyle().Foreground(ui.EclipseCyan).Render("•"), file)
			}
		}

		body += "\n" + ui.HelpStyle.Render("[Enter] Confirmar Commit e Fazer Push • [Esc] Voltar")

	default: // stateMenu
		body = "O que vamos forjar hoje?\n\n"
		if m.message != "" {
			// Define dinamicamente verde para sucesso ou vermelho para erros
			color := "#ff0000"
			if strings.HasPrefix(m.message, "✅") {
				color = "#00ff00"
			}
			body += lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(m.message) + "\n\n"
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

func (m model) generateCommitAction() tea.Cmd {
	return func() tea.Msg {
		err := git.AddAll()
		if err != nil {
			return errMsg(fmt.Errorf("erro ao adicionar arquivos: %v", err))
		}

		diff, err := git.GetDiff()
		if err != nil {
			return errMsg(err)
		}

		if strings.TrimSpace(diff) == "" {
			return errMsg(fmt.Errorf("nenhuma alteração detectada no Git. Garanta que está na raiz de um repositório Git"))
		}

		message, err := ai.GenerateCommitMessage(diff)
		if err != nil {
			return errMsg(err)
		}

		return aiResponseMsg(message)
	}
}

func (m model) executeCommitAndPushAction() tea.Cmd {
	return func() tea.Msg {
		err := git.Commit(strings.TrimSpace(m.aiResult))
		if err != nil {
			return errMsg(fmt.Errorf("falha ao executar commit: %v", err))
		}

		err = git.Push()
		if err != nil {
			return errMsg(fmt.Errorf("commit feito, mas falhou o push: %v. Verifique a sua conexão", err))
		}

		return commitSuccessMsg("sucesso")
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Houve um erro: %v", err)
		os.Exit(1)
	}
}
