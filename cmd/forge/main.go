package main

import (
	"fmt"
	"forge/internal/ai"
	"forge/internal/git"
	"forge/internal/ui"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateMenu state = iota
	stateLoading
	stateResult
	statePushing // Novo estado para o feedback do Commit/Push
)

type model struct {
	choices     []string
	cursor      int
	state       state
	message     string
	aiResult    string
	stagedFiles []string // Lista para armazenar os arquivos alterados
}

func initialModel() model {
	return model{
		choices: []string{"🤖 Forjar Commit (IA)", "📝 Gerar README", "⚙️  Configurações"},
		state:   stateMenu,
	}
}

func (m model) Init() tea.Cmd { return nil }

type aiResponseMsg string
type commitSuccessMsg string // Nova mensagem de sucesso para fechar o app
type errMsg error

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.state != stateMenu && m.state != statePushing {
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
			// Alterado: Ao pressionar Enter na sugestão, inicia o Commit + Push real
			if msg.String() == "enter" {
				m.state = statePushing
				return m, m.executeCommitAndPushAction()
			}
		}

	case aiResponseMsg:
		m.state = stateResult
		m.aiResult = string(msg)
		// Busca a lista de arquivos alterados após o processamento
		files, _ := git.GetStagedFiles()
		m.stagedFiles = files
		return m, nil

	case commitSuccessMsg:
		// Sucesso total, encerra o programa de forma limpa
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
		// Visual de feedback enquanto o Git trabalha em segundo plano
		body = "\n  " + lipgloss.NewStyle().Foreground(ui.EclipseCyan).Render("🚀 Gravando commit e empurrando para o GitHub...")

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

		body += "\n" + ui.HelpStyle.Render("[Enter] Confirmar Commit e Fazer Push • [Esc] Voltar")

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
		// 1. Resolve o git add e captura o erro se ele falhar
		err := git.AddAll()
		if err != nil {
			return errMsg(fmt.Errorf("erro ao adicionar arquivos: %v", err))
		}

		// 2. Extrai o diff das alterações preparadas
		diff, err := git.GetDiff()
		if err != nil {
			return errMsg(err)
		}

		// 3. Validação crucial: Se não houver código alterado, não chama a IA
		if strings.TrimSpace(diff) == "" {
			return errMsg(fmt.Errorf("nenhuma alteração detectada no Git. Altere algum arquivo primeiro"))
		}

		// 4. Chamada real para o nosso motor HTTP leve do Gemini
		message, err := ai.GenerateCommitMessage(diff)
		if err != nil {
			return errMsg(err) // Se a API falhar, o erro REAL vai direto para a tela vermelha
		}

		return aiResponseMsg(message)
	}
}

// executeCommitAndPushAction executa os comandos finais no terminal do SO
func (m model) executeCommitAndPushAction() tea.Cmd {
	return func() tea.Msg {
		// 1. Executa o commit real usando a string limpa sugerida pela IA
		err := git.Commit(strings.TrimSpace(m.aiResult))
		if err != nil {
			return errMsg(fmt.Errorf("falha ao executar commit: %v", err))
		}

		// 2. Dispara para o GitHub/GitLab remotos
		err = git.Push()
		if err != nil {
			return errMsg(fmt.Errorf("commit feito, mas falhou o push: %v. Verifique a sua conexão", err))
		}

		return commitSuccessMsg("sucesso")
	}
}
