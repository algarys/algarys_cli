package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/algarys/algarys_cli/cmd/ui"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

const algarysDir = ".algarys"
const transcribeDir = "transcricao"

// Script Python embutido - prints de status vão para stderr, texto final para stdout
const transcribePyScript = `#!/usr/bin/env python3
"""Script de Transcrição de Áudio usando Whisper."""

import argparse
import os
import sys
from pathlib import Path

import whisper


def transcrever_audio(caminho_audio: str, modelo: str = "large", idioma: str = None) -> str:
    print(f"STATUS:Carregando modelo '{modelo}'...", file=sys.stderr)
    model = whisper.load_model(modelo)

    print(f"STATUS:Transcrevendo áudio...", file=sys.stderr)

    opcoes = {}
    if idioma:
        opcoes["language"] = idioma

    resultado = model.transcribe(caminho_audio, **opcoes)
    return resultado["text"]


def main():
    parser = argparse.ArgumentParser(description="Transcreve áudio para texto usando Whisper")
    parser.add_argument("arquivo", help="Caminho para o arquivo de áudio")
    parser.add_argument("-m", "--modelo", default="large",
                        choices=["tiny", "base", "small", "medium", "large"])
    parser.add_argument("-l", "--idioma", help="Código do idioma (pt, en, es)")
    args = parser.parse_args()

    if not os.path.exists(args.arquivo):
        print(f"ERRO:Arquivo não encontrado: {args.arquivo}", file=sys.stderr)
        sys.exit(1)

    try:
        texto = transcrever_audio(args.arquivo, args.modelo, args.idioma)
        # Texto final vai para stdout (limpo, sem prefixo)
        print(texto)
    except Exception as e:
        print(f"ERRO:{e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
`

const transcribePyProject = `[project]
name = "algarys-transcricao"
version = "1.0.0"
description = "Transcrição de áudio com Whisper - Algarys CLI"
requires-python = ">=3.10"
dependencies = [
    "openai-whisper>=20231117",
    "torch>=2.0.0",
]
`

var (
	transcribeModel string
	transcribeLang  string
)

var transcribeCmd = &cobra.Command{
	Use:   "transcribe <arquivo>",
	Short: "Transcreve arquivos de áudio para texto usando Whisper",
	Long: `Transcreve arquivos de áudio (mp3, wav, m4a, ogg, etc) para texto
usando o modelo Whisper da OpenAI, executado localmente.

Modelos disponíveis:
  tiny    ~39M parâmetros  (mais rápido, menos preciso)
  base    ~74M parâmetros
  small   ~244M parâmetros
  medium  ~769M parâmetros
  large   ~1550M parâmetros (padrão, mais preciso)

Requer: uv, ffmpeg, Python 3.10+`,
	Args: cobra.ExactArgs(1),
	Run:  runTranscribe,
}

func init() {
	transcribeCmd.Flags().StringVarP(&transcribeModel, "model", "m", "large", "Modelo Whisper (tiny, base, small, medium, large)")
	transcribeCmd.Flags().StringVarP(&transcribeLang, "lang", "l", "", "Código do idioma (pt, en, es). Padrão: auto-detectar")
	rootCmd.AddCommand(transcribeCmd)
}

func runTranscribe(cmd *cobra.Command, args []string) {
	audioFile := args[0]

	fmt.Println()
	fmt.Println(ui.RenderBanner())
	fmt.Println()

	subtitle := lipgloss.NewStyle().
		Foreground(ui.TextDim).
		Italic(true).
		Render("  🎧 Transcrição de áudio com Whisper")
	fmt.Println(subtitle)
	fmt.Println()

	// Verificar se arquivo existe
	absAudioFile, err := filepath.Abs(audioFile)
	if err != nil {
		fmt.Println(ui.RenderError(fmt.Sprintf("Caminho inválido: %v", err)))
		os.Exit(1)
	}

	if _, err := os.Stat(absAudioFile); os.IsNotExist(err) {
		fmt.Println(ui.RenderError(fmt.Sprintf("Arquivo não encontrado: %s", audioFile)))
		os.Exit(1)
	}

	// Verificar dependências
	if !checkTranscribeDeps() {
		return
	}

	// Setup do ambiente Python (se necessário)
	projectDir := getTranscribeDir()
	if !isTranscribeSetup(projectDir) {
		if !setupTranscribeEnv(projectDir) {
			return
		}
	}

	// Executar transcrição
	transcribedText := runTranscription(projectDir, absAudioFile)
	if transcribedText == "" {
		return
	}

	// Mostrar texto transcrito
	showTranscribedText(transcribedText)

	// Perguntar se quer salvar
	askToSaveTranscription(transcribedText, absAudioFile)
}

func checkTranscribeDeps() bool {
	deps := []struct {
		cmd     string
		name    string
		install string
	}{
		{"uv", "UV (gerenciador Python)", "curl -LsSf https://astral.sh/uv/install.sh | sh"},
		{"ffmpeg", "FFmpeg (processamento de áudio)", "brew install ffmpeg"},
	}

	for _, dep := range deps {
		if _, err := exec.LookPath(dep.cmd); err != nil {
			fmt.Println(ui.RenderError(fmt.Sprintf("%s não encontrado", dep.name)))
			fmt.Println()
			fmt.Println(lipgloss.NewStyle().Foreground(ui.Muted).PaddingLeft(2).Render("Instale com:"))
			fmt.Println(lipgloss.NewStyle().Foreground(ui.Primary).PaddingLeft(4).Render(dep.install))
			fmt.Println()
			return false
		}
	}

	return true
}

func getTranscribeDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, algarysDir, transcribeDir)
}

func isTranscribeSetup(projectDir string) bool {
	scriptPath := filepath.Join(projectDir, "transcrever.py")
	venvPath := filepath.Join(projectDir, ".venv")

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(venvPath); os.IsNotExist(err) {
		return false
	}

	return true
}

func setupTranscribeEnv(projectDir string) bool {
	fmt.Println(lipgloss.NewStyle().
		Foreground(ui.TextDim).
		PaddingLeft(2).
		Render("Primeira execução - configurando ambiente..."))
	fmt.Println()

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		fmt.Println(ui.RenderError(fmt.Sprintf("Erro ao criar diretório: %v", err)))
		return false
	}

	spinner := ui.NewSpinner(ui.IconFile + "  Criando script de transcrição...")
	spinner.Start()
	time.Sleep(200 * time.Millisecond)

	scriptPath := filepath.Join(projectDir, "transcrever.py")
	if err := os.WriteFile(scriptPath, []byte(transcribePyScript), 0644); err != nil {
		spinner.Error("Erro ao criar script")
		return false
	}

	pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(transcribePyProject), 0644); err != nil {
		spinner.Error("Erro ao criar pyproject.toml")
		return false
	}
	spinner.Success("Script de transcrição criado")

	spinnerDeps := ui.NewSpinner(ui.IconPython + "  Instalando dependências (whisper + torch)...")
	spinnerDeps.Start()

	uvCmd := exec.Command("uv", "sync")
	uvCmd.Dir = projectDir
	uvCmd.Stdout = nil
	uvCmd.Stderr = nil

	if err := uvCmd.Run(); err != nil {
		spinnerDeps.Error("Erro ao instalar dependências")
		fmt.Println()
		fmt.Println(ui.RenderError(fmt.Sprintf("Falha no uv sync: %v", err)))
		fmt.Println(lipgloss.NewStyle().Foreground(ui.Muted).PaddingLeft(2).Render(
			fmt.Sprintf("Tente manualmente: cd %s && uv sync", projectDir),
		))
		return false
	}

	spinnerDeps.Success("Dependências instaladas")
	fmt.Println()
	return true
}

func runTranscription(projectDir, audioFile string) string {
	// Info do arquivo
	fileInfo, _ := os.Stat(audioFile)
	fileName := filepath.Base(audioFile)
	fileSizeMB := float64(fileInfo.Size()) / (1024 * 1024)

	langDisplay := "auto-detectar"
	if transcribeLang != "" {
		langDisplay = transcribeLang
	}

	infoBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.Muted).
		Padding(0, 2).
		Render(
			lipgloss.NewStyle().Foreground(ui.Text).Bold(true).Render("Arquivo: ") +
				lipgloss.NewStyle().Foreground(ui.Primary).Render(fileName) + "\n" +
				lipgloss.NewStyle().Foreground(ui.Text).Bold(true).Render("Tamanho: ") +
				lipgloss.NewStyle().Foreground(ui.TextDim).Render(fmt.Sprintf("%.1f MB", fileSizeMB)) + "\n" +
				lipgloss.NewStyle().Foreground(ui.Text).Bold(true).Render("Modelo:  ") +
				lipgloss.NewStyle().Foreground(ui.Primary).Render(transcribeModel) + "\n" +
				lipgloss.NewStyle().Foreground(ui.Text).Bold(true).Render("Idioma:  ") +
				lipgloss.NewStyle().Foreground(ui.TextDim).Render(langDisplay),
		)
	fmt.Println(infoBox)
	fmt.Println()

	// Montar argumentos (sem -o, a saída vai para stdout)
	uvArgs := []string{"run", "python", "transcrever.py", audioFile, "-m", transcribeModel}
	if transcribeLang != "" {
		uvArgs = append(uvArgs, "-l", transcribeLang)
	}

	// Spinner enquanto transcreve
	spinner := ui.NewSpinner("🎧  Transcrevendo áudio...")
	spinner.Start()

	uvCmd := exec.Command("uv", uvArgs...)
	uvCmd.Dir = projectDir

	// stdout = texto transcrito, stderr = mensagens de status
	stdoutPipe, err := uvCmd.StdoutPipe()
	if err != nil {
		spinner.Error("Erro ao iniciar transcrição")
		return ""
	}

	stderrPipe, err := uvCmd.StderrPipe()
	if err != nil {
		spinner.Error("Erro ao iniciar transcrição")
		return ""
	}

	if err := uvCmd.Start(); err != nil {
		spinner.Error(fmt.Sprintf("Erro ao iniciar: %v", err))
		return ""
	}

	// Ler stderr em background (mensagens de status do Whisper)
	go func() {
		stderrScanner := bufio.NewScanner(stderrPipe)
		for stderrScanner.Scan() {
			line := stderrScanner.Text()
			// Atualizar spinner com mensagens de status
			if strings.HasPrefix(line, "STATUS:") {
				msg := strings.TrimPrefix(line, "STATUS:")
				spinner.Stop()
				spinner = ui.NewSpinner("🎧  " + msg)
				spinner.Start()
			}
		}
	}()

	// Ler stdout (texto transcrito)
	var textBuilder strings.Builder
	stdoutScanner := bufio.NewScanner(stdoutPipe)
	// Buffer maior para textos longos
	buf := make([]byte, 0, 1024*1024)
	stdoutScanner.Buffer(buf, 10*1024*1024)
	for stdoutScanner.Scan() {
		textBuilder.WriteString(stdoutScanner.Text())
		textBuilder.WriteString("\n")
	}

	err = uvCmd.Wait()
	spinner.Stop()

	if err != nil {
		fmt.Println(ui.RenderError("Transcrição falhou"))
		fmt.Println()
		return ""
	}

	text := strings.TrimSpace(textBuilder.String())
	if text == "" {
		fmt.Println(ui.RenderWarning("Nenhum texto detectado no áudio"))
		fmt.Println()
		return ""
	}

	fmt.Println(ui.RenderSuccess("Transcrição concluída!"))
	fmt.Println()
	return text
}

func showTranscribedText(text string) {
	// Título
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.Primary).
		Bold(true).
		PaddingLeft(2)
	fmt.Println(titleStyle.Render("📄 Texto transcrito:"))
	fmt.Println()

	// Caixa com o texto
	textBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.Muted).
		Padding(1, 2).
		Width(80).
		Foreground(ui.Text).
		Render(text)
	fmt.Println(textBox)
	fmt.Println()
}

func askToSaveTranscription(text, audioFile string) {
	reader := bufio.NewReader(os.Stdin)

	// Perguntar se quer salvar
	savePrompt := lipgloss.NewStyle().
		Foreground(ui.Primary).
		Bold(true).
		PaddingLeft(2).
		Render("Deseja salvar em um arquivo de texto?")
	fmt.Print(savePrompt)
	fmt.Print(lipgloss.NewStyle().Foreground(ui.TextDim).Render(" [S/n] "))

	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "" && response != "s" && response != "sim" {
		fmt.Println()
		fmt.Println(lipgloss.NewStyle().
			Foreground(ui.Muted).
			Italic(true).
			PaddingLeft(2).
			Render("Transcrição não salva."))
		fmt.Println()
		return
	}

	// Sugerir nome do arquivo
	ext := filepath.Ext(audioFile)
	suggestedName := strings.TrimSuffix(filepath.Base(audioFile), ext) + ".txt"

	namePrompt := lipgloss.NewStyle().
		Foreground(ui.Primary).
		Bold(true).
		PaddingLeft(2).
		Render("Nome do arquivo:")
	fmt.Println()
	fmt.Print(namePrompt)
	fmt.Print(lipgloss.NewStyle().Foreground(ui.TextDim).Render(fmt.Sprintf(" (%s) ", suggestedName)))

	fileName, _ := reader.ReadString('\n')
	fileName = strings.TrimSpace(fileName)

	// Usar nome sugerido se vazio
	if fileName == "" {
		fileName = suggestedName
	}

	// Adicionar .txt se não tiver extensão
	if !strings.Contains(fileName, ".") {
		fileName = fileName + ".txt"
	}

	// Resolver caminho - salvar no diretório do áudio
	audioDir := filepath.Dir(audioFile)
	outputPath := filepath.Join(audioDir, fileName)

	// Se o usuário passou um caminho absoluto, respeitar
	if filepath.IsAbs(fileName) {
		outputPath = fileName
	}

	// Confirmar
	fmt.Println()
	confirmPrompt := lipgloss.NewStyle().
		Foreground(ui.TextDim).
		PaddingLeft(2).
		Render(fmt.Sprintf("Salvar em: %s", outputPath))
	fmt.Println(confirmPrompt)
	fmt.Print(lipgloss.NewStyle().Foreground(ui.Primary).PaddingLeft(2).Bold(true).Render("Confirmar?"))
	fmt.Print(lipgloss.NewStyle().Foreground(ui.TextDim).Render(" [S/n] "))

	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))

	if confirm != "" && confirm != "s" && confirm != "sim" {
		fmt.Println()
		fmt.Println(lipgloss.NewStyle().
			Foreground(ui.Muted).
			Italic(true).
			PaddingLeft(2).
			Render("Transcrição não salva."))
		fmt.Println()
		return
	}

	// Salvar
	if err := os.WriteFile(outputPath, []byte(text), 0644); err != nil {
		fmt.Println()
		fmt.Println(ui.RenderError(fmt.Sprintf("Erro ao salvar: %v", err)))
		return
	}

	fmt.Println()
	successBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.Primary).
		Padding(1, 2).
		Render(
			lipgloss.NewStyle().Foreground(ui.Primary).Bold(true).Render(
				fmt.Sprintf("%s Arquivo salvo!", ui.IconDone),
			) + "\n\n" +
				lipgloss.NewStyle().Foreground(ui.TextDim).Render("Local: ") +
				lipgloss.NewStyle().Foreground(ui.Primary).Render(outputPath),
		)
	fmt.Println(successBox)
	fmt.Println()
}
