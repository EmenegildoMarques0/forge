package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Cores da Eclipse Dynamics
	EclipseBlue = lipgloss.Color("#5f5fff")
	EclipseCyan = lipgloss.Color("#00ffff")
	White       = lipgloss.Color("#ffffff")
	Gray        = lipgloss.Color("#737373")

	// Estilo da Caixa Principal
	MainBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(EclipseBlue).
		Padding(1, 2).
		Margin(1, 0)

	// Título Principal (Cabeçalho)
	Header = lipgloss.NewStyle().
		Foreground(White).
		Background(EclipseBlue).
		Bold(true).
		Padding(0, 1)

	// --- A VARIÁVEL QUE ESTAVA FALTANDO ---
	SelectedItem = lipgloss.NewStyle().
			Foreground(EclipseCyan).
			Bold(true)

	// Instruções de rodapé
	HelpStyle = lipgloss.NewStyle().Foreground(Gray).Italic(true)
)
