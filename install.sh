#!/bin/bash
set -e

# Cores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}"
echo "  █████╗ ██╗      ██████╗  █████╗ ██████╗ ██╗   ██╗███████╗"
echo " ██╔══██╗██║     ██╔════╝ ██╔══██╗██╔══██╗╚██╗ ██╔╝██╔════╝"
echo " ███████║██║     ██║  ███╗███████║██████╔╝ ╚████╔╝ ███████╗"
echo " ██╔══██║██║     ██║   ██║██╔══██║██╔══██╗  ╚██╔╝  ╚════██║"
echo " ██║  ██║███████╗╚██████╔╝██║  ██║██║  ██║   ██║   ███████║"
echo " ╚═╝  ╚═╝╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝   ╚══════╝"
echo -e "${NC}"
echo "Instalador do Algarys CLI"
echo ""

# Detectar OS e arquitetura
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo -e "${RED}Arquitetura não suportada: $ARCH${NC}"; exit 1 ;;
esac

case "$OS" in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    *) echo -e "${RED}Sistema operacional não suportado: $OS${NC}"; exit 1 ;;
esac

echo -e "${YELLOW}→ Detectado: ${OS}/${ARCH}${NC}"

# Buscar última versão (usando gh CLI para repos privados)
echo -e "${YELLOW}→ Buscando última versão...${NC}"

# Verificar se gh está disponível e autenticado
if command -v gh &> /dev/null; then
    LATEST_VERSION=$(gh release view --repo algarys/algarys_cli --json tagName -q '.tagName' 2>/dev/null)
fi

# Fallback para API pública (repos públicos)
if [ -z "$LATEST_VERSION" ]; then
    LATEST_VERSION=$(curl -s https://api.github.com/repos/algarys/algarys_cli/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
fi

if [ -z "$LATEST_VERSION" ]; then
    echo -e "${RED}Erro ao buscar versão.${NC}"
    echo -e "${YELLOW}Para repos privados, instale e autentique o gh CLI:${NC}"
    echo -e "  brew install gh && gh auth login"
    exit 1
fi

echo -e "${GREEN}  ✓ Versão: ${LATEST_VERSION}${NC}"

# Download
INSTALL_DIR="/usr/local/bin"
TMP_DIR=$(mktemp -d)

echo -e "${YELLOW}→ Baixando algarys_${OS}_${ARCH}.tar.gz...${NC}"

# Usar gh CLI para download (repos privados)
if command -v gh &> /dev/null; then
    if ! gh release download "$LATEST_VERSION" --repo algarys/algarys_cli --pattern "algarys_${OS}_${ARCH}.tar.gz" --dir "$TMP_DIR" 2>/dev/null; then
        echo -e "${RED}Erro ao baixar. Verifique se a release existe.${NC}"
        rm -rf "$TMP_DIR"
        exit 1
    fi
    mv "$TMP_DIR/algarys_${OS}_${ARCH}.tar.gz" "$TMP_DIR/algarys.tar.gz"
else
    # Fallback para curl (repos públicos)
    DOWNLOAD_URL="https://github.com/algarys/algarys_cli/releases/download/${LATEST_VERSION}/algarys_${OS}_${ARCH}.tar.gz"
    if ! curl -sL "$DOWNLOAD_URL" -o "$TMP_DIR/algarys.tar.gz"; then
        echo -e "${RED}Erro ao baixar. Verifique se a release existe.${NC}"
        rm -rf "$TMP_DIR"
        exit 1
    fi
fi

# Extrair
echo -e "${YELLOW}→ Extraindo...${NC}"
tar -xzf "$TMP_DIR/algarys.tar.gz" -C "$TMP_DIR"

# Instalar
echo -e "${YELLOW}→ Instalando em ${INSTALL_DIR}...${NC}"
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_DIR/algarys" "$INSTALL_DIR/algarys"
else
    sudo mv "$TMP_DIR/algarys" "$INSTALL_DIR/algarys"
fi
chmod +x "$INSTALL_DIR/algarys"

# Limpar
rm -rf "$TMP_DIR"

# Verificar instalação
if command -v algarys &> /dev/null; then
    echo ""
    echo -e "${GREEN}✓ Algarys CLI instalado com sucesso!${NC}"
    echo ""
    algarys version
    echo ""
    echo -e "Execute ${CYAN}algarys init${NC} para criar um novo projeto."
else
    echo -e "${RED}Erro: instalação falhou${NC}"
    exit 1
fi
