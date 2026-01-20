package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Spinner frames
var (
	DotsSpinner  = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	LineSpinner  = []string{"-", "\\", "|", "/"}
	GrowSpinner  = []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃", "▂"}
	PulseSpinner = []string{"●", "○", "●", "○"}
	ArrowSpinner = []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"}
)

type Spinner struct {
	frames   []string
	index    int
	message  string
	done     chan bool
	style    lipgloss.Style
	interval time.Duration
}

func NewSpinner(message string) *Spinner {
	return &Spinner{
		frames:   DotsSpinner,
		message:  message,
		done:     make(chan bool),
		style:    lipgloss.NewStyle().Foreground(Primary),
		interval: 80 * time.Millisecond,
	}
}

func (s *Spinner) Start() {
	go func() {
		for {
			select {
			case <-s.done:
				return
			default:
				frame := s.style.Render(s.frames[s.index])
				fmt.Printf("\r%s %s", frame, s.message)
				s.index = (s.index + 1) % len(s.frames)
				time.Sleep(s.interval)
			}
		}
	}()
}

func (s *Spinner) Stop() {
	s.done <- true
	fmt.Print("\r\033[K") // Limpa a linha
}

func (s *Spinner) Success(message string) {
	s.done <- true
	fmt.Print("\r\033[K")
	fmt.Println(RenderSuccess(message))
}

func (s *Spinner) Error(message string) {
	s.done <- true
	fmt.Print("\r\033[K")
	fmt.Println(RenderError(message))
}

func (s *Spinner) Warning(message string) {
	s.done <- true
	fmt.Print("\r\033[K")
	fmt.Println(RenderWarning(message))
}

// Função helper para executar com spinner
func WithSpinner(message string, fn func() error) error {
	spinner := NewSpinner(message)
	spinner.Start()

	err := fn()

	if err != nil {
		spinner.Error(message + " - falhou")
		return err
	}

	spinner.Success(message)
	return nil
}

// Progress bar simples
type ProgressBar struct {
	total   int
	current int
	width   int
	style   lipgloss.Style
}

func NewProgressBar(total, width int) *ProgressBar {
	return &ProgressBar{
		total: total,
		width: width,
		style: lipgloss.NewStyle().Foreground(Primary),
	}
}

func (p *ProgressBar) Increment() {
	p.current++
	p.render()
}

func (p *ProgressBar) render() {
	percentage := float64(p.current) / float64(p.total)
	filled := int(percentage * float64(p.width))
	empty := p.width - filled

	bar := p.style.Render(repeatString("█", filled)) +
		   MutedStyle.Render(repeatString("░", empty))

	fmt.Printf("\r%s %3.0f%%", bar, percentage*100)

	if p.current >= p.total {
		fmt.Println()
	}
}

func repeatString(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
