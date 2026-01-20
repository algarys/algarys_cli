package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Cores da marca Algarys
var (
	Primary   = lipgloss.Color("#01FFCE") // Verde-água principal
	Secondary = lipgloss.Color("#00D9B1") // Verde-água escuro
	Accent    = lipgloss.Color("#7B61FF") // Roxo accent
	Success   = lipgloss.Color("#01FFCE")
	Warning   = lipgloss.Color("#FFB800")
	Error     = lipgloss.Color("#FF5757")
	Muted     = lipgloss.Color("#6B7280")
	Text      = lipgloss.Color("#FFFFFF")
	TextDim   = lipgloss.Color("#9CA3AF")
)

// Estilos de texto
var (
	// Títulos
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(TextDim)

	// Status
	SuccessStyle = lipgloss.NewStyle().
			Foreground(Success)

	WarningStyle = lipgloss.NewStyle().
			Foreground(Warning)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Error)

	MutedStyle = lipgloss.NewStyle().
			Foreground(Muted)

	// Destaques
	HighlightStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary)

	AccentStyle = lipgloss.NewStyle().
			Foreground(Accent)

	// Código/Comandos
	CodeStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Background(lipgloss.Color("#1a1a2e")).
			Padding(0, 1)
)

// Estilos de containers
var (
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2)

	InfoBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Muted).
			Padding(1, 2)
)

// Banner da Algarys
const Banner = `
   ▄▀█ █░░ █▀▀ ▄▀█ █▀█ █▄█ █▀
   █▀█ █▄▄ █▄█ █▀█ █▀▄ ░█░ ▄█
`

const BannerAlt = `
  ╔═╗╦  ╔═╗╔═╗╦═╗╦ ╦╔═╗
  ╠═╣║  ║ ╦╠═╣╠╦╝╚╦╝╚═╗
  ╩ ╩╩═╝╚═╝╩ ╩╩╚═ ╩ ╚═╝
`

const BannerLarge = `
    ___    __    _________    ____  __  _______
   /   |  / /   / ____/   |  / __ \/ / / / ___/
  / /| | / /   / / __/ /| | / /_/ / /_/ /\__ \
 / ___ |/ /___/ /_/ / ___ |/ _, _/ __  /___/ /
/_/  |_/_____/\____/_/  |_/_/ |_/_/ /_//____/
`

// Ícones/Emojis
const (
	IconSuccess  = "✓"
	IconError    = "✗"
	IconWarning  = "⚠"
	IconInfo     = "ℹ"
	IconArrow    = "→"
	IconDot      = "●"
	IconCheck    = "✔"
	IconX        = "✘"
	IconStar     = "★"
	IconRocket   = "🚀"
	IconPackage  = "📦"
	IconFolder   = "📁"
	IconFile     = "📄"
	IconGit      = "🔀"
	IconGitHub   = "🐙"
	IconPython   = "🐍"
	IconAI       = "🤖"
	IconMagic    = "✨"
	IconLock     = "🔒"
	IconKey      = "🔑"
	IconGear     = "⚙️"
	IconClock    = "⏱️"
	IconDone     = "🎉"
)

// Funções helper
func RenderBanner() string {
	return lipgloss.NewStyle().
		Foreground(Primary).
		Bold(true).
		Render(BannerLarge)
}

func RenderStep(icon, message string) string {
	return lipgloss.NewStyle().
		Foreground(Primary).
		Render(icon+" ") + message
}

func RenderSuccess(message string) string {
	return SuccessStyle.Render(IconSuccess+" ") + message
}

func RenderError(message string) string {
	return ErrorStyle.Render(IconError+" ") + message
}

func RenderWarning(message string) string {
	return WarningStyle.Render(IconWarning+" ") + message
}

func RenderInfo(message string) string {
	return AccentStyle.Render(IconInfo+" ") + message
}

func RenderCommand(cmd string) string {
	return CodeStyle.Render(cmd)
}

func RenderHighlight(text string) string {
	return HighlightStyle.Render(text)
}

func RenderBox(title, content string) string {
	titleRendered := TitleStyle.Render(title)
	return BoxStyle.Render(titleRendered + "\n\n" + content)
}
