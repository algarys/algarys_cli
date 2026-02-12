# Algarys CLI - Instalador Windows
$ErrorActionPreference = "Stop"

Write-Host ""
Write-Host "  ALGARYS CLI" -ForegroundColor Cyan
Write-Host "  Instalador Windows" -ForegroundColor Cyan
Write-Host ""

# Detectar arquitetura
$arch = if ([System.Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
} else {
    Write-Host "Erro: sistema 32-bit nao suportado." -ForegroundColor Red
    exit 1
}

Write-Host "-> Detectado: windows/$arch" -ForegroundColor Yellow

# Buscar ultima versao
Write-Host "-> Buscando ultima versao..." -ForegroundColor Yellow

$latestVersion = $null

# Tentar com gh CLI
if (Get-Command gh -ErrorAction SilentlyContinue) {
    try {
        $latestVersion = gh release view --repo algarys/algarys_cli --json tagName -q '.tagName' 2>$null
    } catch {}
}

# Fallback para API publica
if (-not $latestVersion) {
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/algarys/algarys_cli/releases/latest" -UseBasicParsing
        $latestVersion = $release.tag_name
    } catch {
        Write-Host "Erro ao buscar versao. Verifique sua conexao." -ForegroundColor Red
        exit 1
    }
}

if (-not $latestVersion) {
    Write-Host "Erro ao buscar versao." -ForegroundColor Red
    exit 1
}

Write-Host "  Versao: $latestVersion" -ForegroundColor Green

# Diretorio de instalacao
$installDir = "$env:LOCALAPPDATA\algarys"
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

$tmpDir = Join-Path $env:TEMP "algarys_install_$(Get-Random)"
New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null

$zipFile = Join-Path $tmpDir "algarys.zip"
$fileName = "algarys_windows_$arch.zip"

# Download
Write-Host "-> Baixando $fileName..." -ForegroundColor Yellow

$downloaded = $false

# Tentar com gh CLI
if (Get-Command gh -ErrorAction SilentlyContinue) {
    try {
        gh release download $latestVersion --repo algarys/algarys_cli --pattern $fileName --dir $tmpDir 2>$null
        $downloadedFile = Join-Path $tmpDir $fileName
        if (Test-Path $downloadedFile) {
            Move-Item $downloadedFile $zipFile -Force
            $downloaded = $true
        }
    } catch {}
}

# Fallback para download direto
if (-not $downloaded) {
    try {
        $downloadUrl = "https://github.com/algarys/algarys_cli/releases/download/$latestVersion/$fileName"
        Invoke-WebRequest -Uri $downloadUrl -OutFile $zipFile -UseBasicParsing
        $downloaded = $true
    } catch {
        Write-Host "Erro ao baixar. Verifique se a release existe." -ForegroundColor Red
        Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
        exit 1
    }
}

# Extrair
Write-Host "-> Extraindo..." -ForegroundColor Yellow
Expand-Archive -Path $zipFile -DestinationPath $tmpDir -Force

# Instalar
Write-Host "-> Instalando em $installDir..." -ForegroundColor Yellow
$exePath = Join-Path $tmpDir "algarys.exe"
if (-not (Test-Path $exePath)) {
    Write-Host "Erro: binario nao encontrado no arquivo." -ForegroundColor Red
    Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
    exit 1
}

Copy-Item $exePath (Join-Path $installDir "algarys.exe") -Force

# Limpar
Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue

# Adicionar ao PATH se necessario
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    Write-Host "-> Adicionando ao PATH..." -ForegroundColor Yellow
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    $env:Path = "$env:Path;$installDir"
    Write-Host "  PATH atualizado" -ForegroundColor Green
}

# Verificar instalacao
Write-Host ""
$algarysBin = Join-Path $installDir "algarys.exe"
if (Test-Path $algarysBin) {
    Write-Host "Algarys CLI instalado com sucesso!" -ForegroundColor Green
    Write-Host ""
    & $algarysBin version
    Write-Host ""
    Write-Host "Execute 'algarys init' para criar um novo projeto." -ForegroundColor Cyan
    Write-Host ""
    Write-Host "IMPORTANTE: Feche e abra o terminal para o comando 'algarys' funcionar." -ForegroundColor Yellow
} else {
    Write-Host "Erro: instalacao falhou" -ForegroundColor Red
    exit 1
}
