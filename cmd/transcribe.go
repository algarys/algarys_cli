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

// Script Python embutido
const transcribePyScript = `#!/usr/bin/env python3
"""Script de Transcrição de Áudio usando Whisper."""

import argparse
import os
import sys
from pathlib import Path

import whisper


def transcrever_audio(caminho_audio: str, modelo: str = "large", idioma: str = None) -> str:
    print(f"🔄 Carregando modelo '{modelo}'...")
    model = whisper.load_model(modelo)

    print(f"🎧 Transcrevendo: {caminho_audio}")

    opcoes = {}
    if idioma:
        opcoes["language"] = idioma
        print(f"📝 Idioma definido: {idioma}")

    resultado = model.transcribe(caminho_audio, **opcoes)
    return resultado["text"]


def salvar_transcricao(texto: str, caminho_saida: str) -> None:
    with open(caminho_saida, "w", encoding="utf-8") as f:
        f.write(texto)
    print(f"✅ Transcrição salva em: {caminho_saida}")


def main():
    parser = argparse.ArgumentParser(description="Transcreve áudio para texto usando Whisper")
    parser.add_argument("arquivo", help="Caminho para o arquivo de áudio")
    parser.add_argument("-o", "--output", help="Caminho do arquivo de saída")
    parser.add_argument("-m", "--modelo", default="large",
                        choices=["tiny", "base", "small", "medium", "large"])
    parser.add_argument("-l", "--idioma", help="Código do idioma (pt, en, es)")
    args = parser.parse_args()

    if not os.path.exists(args.arquivo):
        print(f"❌ Erro: Arquivo não encontrado: {args.arquivo}")
        sys.exit(1)

    if args.output:
        caminho_saida = args.output
    else:
        caminho_saida = str(Path(args.arquivo).with_suffix(".txt"))

    try:
        texto = transcrever_audio(args.arquivo, args.modelo, args.idioma)
        salvar_transcricao(texto, caminho_saida)

        print("\n📄 Prévia da transcrição:")
        print("-" * 50)
        preview = texto[:500] + "..." if len(texto) > 500 else texto
        print(preview)
        print("-" * 50)
    except Exception as e:
        print(f"❌ Erro durante a transcrição: {e}")
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
	transcribeModel  string
	transcribeLang   string
	transcribeOutput string
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
	transcribeCmd.Flags().StringVarP(&transcribeOutput, "output", "o", "", "Arquivo de saída (padrão: <nome>.txt)")
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
	runTranscription(projectDir, absAudioFile)
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
	// Verificar se o script e o .venv existem
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

	// Criar diretório
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		fmt.Println(ui.RenderError(fmt.Sprintf("Erro ao criar diretório: %v", err)))
		return false
	}

	// Escrever script Python
	spinner := ui.NewSpinner(ui.IconFile + "  Criando script de transcrição...")
	spinner.Start()
	time.Sleep(200 * time.Millisecond)

	scriptPath := filepath.Join(projectDir, "transcrever.py")
	if err := os.WriteFile(scriptPath, []byte(transcribePyScript), 0644); err != nil {
		spinner.Error("Erro ao criar script")
		return false
	}

	// Escrever pyproject.toml
	pyprojectPath := filepath.Join(projectDir, "pyproject.toml")
	if err := os.WriteFile(pyprojectPath, []byte(transcribePyProject), 0644); err != nil {
		spinner.Error("Erro ao criar pyproject.toml")
		return false
	}
	spinner.Success("Script de transcrição criado")

	// Instalar dependências com uv
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

func runTranscription(projectDir, audioFile string) {
	// Informações do arquivo
	fileInfo, _ := os.Stat(audioFile)
	fileName := filepath.Base(audioFile)
	fileSizeMB := float64(fileInfo.Size()) / (1024 * 1024)

	infoBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.Muted).
		Padding(0, 2).
		Render(
			lipgloss.NewStyle().Foreground(ui.Text).Bold(true).Render("Arquivo: ") +
				lipgloss.NewStyle().Foreground(ui.Primary).Render(fileName) + "\n" +
				lipgloss.NewStyle().Foreground(ui.Text).Bold(true).Render("Tamanho: ") +
				lipgloss.NewStyle().Foreground(ui.TextDim).Render(fmt.Sprintf("%.1f MB", fileSizeMB)) + "\n" +
				lipgloss.NewStyle().Foreground(ui.Text).Bold(true).Render("Modelo: ") +
				lipgloss.NewStyle().Foreground(ui.Primary).Render(transcribeModel) + "\n" +
				lipgloss.NewStyle().Foreground(ui.Text).Bold(true).Render("Idioma: ") +
				lipgloss.NewStyle().Foreground(ui.TextDim).Render(func() string {
				if transcribeLang != "" {
					return transcribeLang
				}
				return "auto-detectar"
			}()),
		)
	fmt.Println(infoBox)
	fmt.Println()

	// Montar argumentos
	uvArgs := []string{"run", "python", "transcrever.py", audioFile, "-m", transcribeModel}
	if transcribeLang != "" {
		uvArgs = append(uvArgs, "-l", transcribeLang)
	}
	if transcribeOutput != "" {
		absOutput, _ := filepath.Abs(transcribeOutput)
		uvArgs = append(uvArgs, "-o", absOutput)
	}

	// Executar
	uvCmd := exec.Command("uv", uvArgs...)
	uvCmd.Dir = projectDir

	// Capturar output em tempo real
	stdout, err := uvCmd.StdoutPipe()
	if err != nil {
		fmt.Println(ui.RenderError(fmt.Sprintf("Erro: %v", err)))
		return
	}
	uvCmd.Stderr = uvCmd.Stdout

	if err := uvCmd.Start(); err != nil {
		fmt.Println(ui.RenderError(fmt.Sprintf("Erro ao iniciar transcrição: %v", err)))
		return
	}

	// Ler output linha a linha
	scanner := bufio.NewScanner(stdout)
	outputLines := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		outputLines = append(outputLines, line)

		// Mostrar output estilizado
		styledLine := lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(ui.TextDim).
			Render(line)
		fmt.Println(styledLine)
	}

	err = uvCmd.Wait()
	fmt.Println()

	if err != nil {
		fmt.Println(ui.RenderError("Transcrição falhou"))
		fmt.Println()
		return
	}

	// Determinar arquivo de saída
	outputFile := transcribeOutput
	if outputFile == "" {
		ext := filepath.Ext(audioFile)
		outputFile = strings.TrimSuffix(audioFile, ext) + ".txt"
	}

	// Sucesso
	successBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.Primary).
		Padding(1, 2).
		Render(
			lipgloss.NewStyle().Foreground(ui.Primary).Bold(true).Render(
				fmt.Sprintf("%s Transcrição concluída!", ui.IconDone),
			) + "\n\n" +
				lipgloss.NewStyle().Foreground(ui.TextDim).Render("Arquivo salvo em: ") +
				lipgloss.NewStyle().Foreground(ui.Primary).Render(outputFile),
		)
	fmt.Println(successBox)
	fmt.Println()
}
