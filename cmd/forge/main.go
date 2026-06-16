package main

import (
	"fmt"
	"forge/internal/ai"
	"forge/internal/config"
	"forge/internal/git"
	"forge/internal/ui"
	"os"
	"os/exec"
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
	statePR     // Estado para criação de Pull Request
)

type model struct {
	choices     []string
	cursor      int
	state       state
	message     string
	aiResult    string
	stagedFiles []string
	textInput   textinput.Model // Componente para manipulação de digitação
	prLink      string          // Link da Pull Request criada
	prStep      string          // Mensagem de progresso da criação de PR
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
		choices:   []string{"🤖 Forjar Commit (IA)", "📝 Gerar README", "🔀 Criar Pull Request", "⚙️  Configurações"},
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
			if m.state != stateMenu && m.state != statePushing && m.state != statePR {
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
				m.message = "✅ A Configuração foi gravada com sucesso!"
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
				} else if m.cursor == 2 { // Selecionou a opção "🔀 Criar Pull Request"
					m.state = statePR
					m.message = ""
					m.prStep = "🔍 A verificar GitHub CLI..."
					return m, m.createPRAction()
				} else if m.cursor == 3 { // Selecionou a opção "⚙️ Configurações"
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

	case prStepMsg:
		m.prStep = string(msg)
		return m, nil

	case prSuccessMsg:
		m.state = stateResult
		m.message = msg.message
		m.prLink = msg.link
		m.aiResult = "Pull Request criada com sucesso!"
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

	case statePushing:
		body = "\n  " + lipgloss.NewStyle().Foreground(ui.EclipseCyan).Render("🚀 Gravando commit e empurrando para o GitHub...")

	case statePR:
		body = "\n  " + lipgloss.NewStyle().Foreground(ui.EclipseCyan).Render("🔀 Criando Pull Request no GitHub...")
		if m.prStep != "" {
			body += "\n\n  " + lipgloss.NewStyle().Foreground(lipgloss.Color("#ffff00")).Render(m.prStep)
		}

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

		// Exibe o link da PR se estiver disponível
		if m.prLink != "" {
			body += "\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")).Bold(true).Render("🔗 Pull Request:") + "\n"
			body += lipgloss.NewStyle().Foreground(ui.EclipseCyan).Underline(true).Render(m.prLink) + "\n"
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

// prStepMsg representa uma mensagem de progresso durante a criação de PR
type prStepMsg string

func (m model) createPRAction() tea.Cmd {
	return func() tea.Msg {
		// Verifica se o gh CLI está instalado
		if err := git.CheckGHCLI(); err != nil {
			return errMsg(fmt.Errorf("GitHub CLI (gh) não encontrado. Instale-o para criar Pull Requests automaticamente"))
		}

		// Obtém a branch atual
		currentBranch, err := git.GetCurrentBranch()
		if err != nil {
			return errMsg(fmt.Errorf("erro ao obter branch atual: %v", err))
		}

		// Adiciona todas as alterações ao stage
		if err := git.AddAll(); err != nil {
			return errMsg(fmt.Errorf("erro ao adicionar arquivos: %v", err))
		}

		// Gera dados estruturados para PR a partir do diff staged
		diff, err := git.GetDiff()
		if err != nil {
			return errMsg(fmt.Errorf("erro ao obter diff: %v", err))
		}

		// Se não houver diff staged, tenta obter diff unstaged (arquivos modificados mas não adicionados)
		if strings.TrimSpace(diff) == "" {
			diffUnstaged, err := git.GetUnstagedDiff()
			if err != nil {
				return errMsg(fmt.Errorf("erro ao obter diff unstaged: %v", err))
			}
			diff = diffUnstaged
		}

		if strings.TrimSpace(diff) == "" {
			return errMsg(fmt.Errorf("nenhuma alteração detectada no Git. Garanta que está na raiz de um repositório Git"))
		}

		// Atualiza o passo atual na UI
		m.prStep = "🤖 A gerar dados da Pull Request com IA..."

		prData, err := ai.GeneratePRData(diff)
		if err != nil {
			return errMsg(fmt.Errorf("erro ao gerar dados para PR: %v", err))
		}

		// Atualiza o passo atual na UI
		m.prStep = fmt.Sprintf("🌿 A criar branch '%s'...", prData.BranchName)

		// Cria a nova branch e move as alterações para ela
		if err := git.CreateAndSwitchBranch(prData.BranchName); err != nil {
			return errMsg(fmt.Errorf("falha ao criar branch '%s': %v", prData.BranchName, err))
		}

		// Atualiza o passo atual na UI
		m.prStep = "📝 A fazer commit das alterações..."

		// Faz commit das alterações na nova branch
		if err := git.Commit(prData.Title); err != nil {
			// Volta para a branch original em caso de erro
			git.SwitchBranch(currentBranch)
			return errMsg(fmt.Errorf("falha ao fazer commit na nova branch: %v", err))
		}

		// Atualiza o passo atual na UI
		m.prStep = fmt.Sprintf("🚀 A fazer push da branch '%s' para o remoto...", prData.BranchName)

		// Faz push da nova branch para o remoto
		if err := git.Push(); err != nil {
			git.SwitchBranch(currentBranch)
			return errMsg(fmt.Errorf("falha ao fazer push da branch '%s': %v. Verifique a sua conexão", prData.BranchName, err))
		}

		// Atualiza o passo atual na UI
		m.prStep = "🔀 A criar Pull Request no GitHub..."

		// Cria a Pull Request usando o gh CLI
		if err := git.CreatePullRequest(prData.Title, prData.Body, prData.BranchName, currentBranch); err != nil {
			git.SwitchBranch(currentBranch)
			return errMsg(fmt.Errorf("falha ao criar Pull Request: %v", err))
		}

		// Atualiza o passo atual na UI
		m.prStep = "✅ Pull Request criada com sucesso!"

		// Volta para a branch original
		if err := git.SwitchBranch(currentBranch); err != nil {
			return errMsg(fmt.Errorf("PR criada, mas falhou ao voltar para a branch '%s': %v", currentBranch, err))
		}

		// Tenta obter o link da PR criada
		prLink := getPRLink(prData.BranchName)
		return prSuccessMsg{message: "Pull Request criada com sucesso!", link: prLink}
	}
}

// getPRLink tenta obter o link da PR criada usando o gh CLI
func getPRLink(branchName string) string {
	cmd := exec.Command("gh", "pr", "view", branchName, "--json", "url", "--jq", ".url")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// prSuccessMsg representa uma mensagem de sucesso na criação de PR com link
type prSuccessMsg struct {
	message string
	link    string
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Houve um erro: %v", err)
		os.Exit(1)
	}
}
