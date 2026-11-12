<#
.SYNOPSIS
  一键切换知识图谱数据源路径（MOOCCube）。

.DESCRIPTION
  修改 backend/configs/<Config> 里 knowledge.source_dir 的值，并在首次修改前保留 .bak 备份。
  数据目录是平台本地的外部数据（默认 backend/data/MOOCCube），不指向 engine 仓库；
  换目录只需重新执行本脚本，不需要改任何代码。

  数据源目录应包含：
    relations/prerequisite-dependency.json
    relations/concept-field.json
    relations/course-concept.json
    entities/concept.json
    entities/course.json

.PARAMETER Path
  新的 MOOCCube 数据目录。建议相对 backend/ 运行目录（如 data/MOOCCube），
  也可以给绝对路径。

.PARAMETER Config
  要修改的配置文件，默认 backend/configs/config.local.yaml。

.PARAMETER Show
  只显示当前配置，不修改。

.EXAMPLE
  .\scripts\set-kg-source.ps1 -Show
  .\scripts\set-kg-source.ps1 -Path 'D:\data\MOOCCube'
  .\scripts\set-kg-source.ps1 -Path '..\backend\data\MOOCCube'
#>
param(
  [string]$Path,
  [string]$Config = 'backend/configs/config.local.yaml',
  [switch]$Show
)

$ErrorActionPreference = 'Stop'

$RepoRoot = Split-Path -Parent $PSScriptRoot
if (-not [System.IO.Path]::IsPathRooted($Config)) {
  $Config = Join-Path $RepoRoot $Config
}
if (-not (Test-Path $Config)) {
  throw "配置文件不存在: $Config"
}

$lines = Get-Content $Config

function Get-CurrentValue {
  $inBlock = $false
  for ($i = 0; $i -lt $lines.Count; $i++) {
    if ($lines[$i] -match '^knowledge:\s*$') { $inBlock = $true; continue }
    if ($inBlock -and $lines[$i] -match '^\S') { $inBlock = $false }
    if ($inBlock -and $lines[$i] -match '^\s+source_dir:\s*(.+?)\s*$') {
      return $Matches[1].Trim('"', "'")
    }
  }
  return $null
}

if ($Show) {
  $current = Get-CurrentValue
  if ($current) { Write-Output "当前 knowledge.source_dir: $current" }
  else { Write-Output "当前未配置 knowledge.source_dir（配置文件: $Config）" }
  return
}

if ([string]::IsNullOrWhiteSpace($Path)) {
  throw "请提供 -Path，或用 -Show 查看当前配置"
}

# 允许传相对 backend/ 的路径；先按当前 shell 的 cwd 校验，失败再按 backend/ 解析
$check = $Path
if (-not (Test-Path $check)) {
  $check = Join-Path (Join-Path $RepoRoot 'backend') $Path
}
if (-not (Test-Path $check)) {
  throw "数据目录不存在: $Path（也尝试过 backend/$Path）"
}
$required = 'relations\prerequisite-dependency.json'
if (-not (Test-Path (Join-Path $check $required))) {
  throw "目录里缺少 $required，请确认指向 MOOCCube 根目录"
}

# YAML 里统一用正斜杠，避免反斜杠转义问题
$value = $Path.Replace('\', '/')

# 备份（只备份一次，避免多次切换把原始配置覆盖掉）
$backup = "$Config.bak"
if (-not (Test-Path $backup)) {
  Copy-Item $Config $backup
  Write-Output "已备份原配置: $backup"
}

$newLines = New-Object System.Collections.Generic.List[string]
$inBlock = $false
$found = $false
$inserted = $false

foreach ($line in $lines) {
  if ($line -match '^knowledge:\s*$') {
    $inBlock = $true
    $newLines.Add($line)
    continue
  }
  if ($inBlock -and $line -match '^\S') {
    if (-not $found) {
      $newLines.Add("  source_dir: `"$value`"")
      $inserted = $true
      $found = $true
    }
    $inBlock = $false
  }
  if ($inBlock -and $line -match '^(\s+)source_dir:') {
    $newLines.Add("  source_dir: `"$value`"")
    $found = $true
    $inserted = $true
    continue
  }
  $newLines.Add($line)
}

if (-not $found -and $inBlock) {
  # knowledge: 是文件最后一段
  $newLines.Add("  source_dir: `"$value`"")
  $found = $true
  $inserted = $true
}
if (-not $found) {
  $newLines.Add('')
  $newLines.Add('# 知识图谱数据源（平台本地外部数据，或用 scripts/set-kg-source.ps1 切换）')
  $newLines.Add('knowledge:')
  $newLines.Add("  source_dir: `"$value`"")
  $inserted = $true
}

if (-not $inserted) { throw '未能写入 source_dir，请手工检查配置结构' }

Set-Content -Path $Config -Value $newLines -Encoding UTF8
Write-Output "已更新 knowledge.source_dir -> $value"
Write-Output "配置文件: $Config"
Write-Output "下一步: cd backend; CONFIG_PATH=configs/config.local.yaml go run ./cmd/seed_knowledge -dry-run"

