<#
.SYNOPSIS
  一键重建知识图谱（学科层 + MOOCCube 概念层）。

.DESCRIPTION
  默认先清空上一版图谱派生数据再重建（-KeepOld 可保留旧数据做增量）。
  只清理「知识图谱」相关表：学科、概念、两层先修边、资源关联；
  资源、用户、行为、评分等业务数据不受影响，推荐主链路不会被打断。

  图谱规则（关键词、数据源路径）调整后，重新执行本脚本即可整体刷新。

.PARAMETER Source
  MOOCCube 数据目录；不传则用配置 knowledge.source_dir。

.PARAMETER Config
  相对 backend/ 的配置文件，默认 configs/config.local.yaml。

.PARAMETER DryRun
  只统计，不写库。

.PARAMETER KeepOld
  不做全量清理，直接在旧数据上增量 upsert（旧关联不会消失，一般用于调试）。

.PARAMETER SkipConcepts
  只重建学科层，不导入概念图。

.EXAMPLE
  .\scripts\rebuild-kg.ps1
  .\scripts\rebuild-kg.ps1 -DryRun
  .\scripts\rebuild-kg.ps1 -Source 'D:\data\MOOCCube'
#>
param(
  [string]$Source,
  [string]$Config = 'configs/config.local.yaml',
  [switch]$DryRun,
  [switch]$KeepOld,
  [switch]$SkipConcepts
)

$ErrorActionPreference = 'Stop'

$RepoRoot = Split-Path -Parent $PSScriptRoot
$BackendDir = Join-Path $RepoRoot 'backend'
if (-not (Test-Path $BackendDir)) { throw "找不到 backend 目录: $BackendDir" }

$env:GOCACHE = if ($env:GOCACHE) { $env:GOCACHE } else { Join-Path $RepoRoot '.gocache' }
$env:CONFIG_PATH = $Config

$goArgs = @('run', './cmd/seed_knowledge')
if (-not $KeepOld) { $goArgs += '-reset' }
if ($DryRun) { $goArgs += '-dry-run' }
if ($SkipConcepts) { $goArgs += '-skip-concepts' }
if ($Source) { $goArgs += @('-source', $Source) }

Write-Output "[rebuild-kg] CONFIG_PATH=$Config"
Write-Output "[rebuild-kg] go $($goArgs -join ' ')"

Push-Location $BackendDir
try {
  & go @goArgs
  if ($LASTEXITCODE -ne 0) { throw "重建失败（exit $LASTEXITCODE）" }
} finally {
  Pop-Location
}

Write-Output "[rebuild-kg] 完成。查看接口: GET /api/v1/knowledge-points"

