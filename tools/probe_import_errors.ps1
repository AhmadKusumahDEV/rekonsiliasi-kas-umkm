#!/usr/bin/env pwsh
# Memetakan pesan error impor. Jalankan: pwsh tools/probe_import_errors.ps1
param(
    [string]$Base = "https://temukas.madgo.my.id",
    [string]$Root = ""
)

$ErrorActionPreference = "Stop"
if ([string]::IsNullOrWhiteSpace($Root)) {
    $Root = Join-Path (Split-Path -Parent $PSScriptRoot) (Join-Path "docs" "demo-files")
}

$stamp = Get-Date -Format "yyyyMMddHHmmss"
$email = "demo.errors.$stamp@temukas.test"
$password = "DemoErrors!2026"

$session = $null
$null = Invoke-RestMethod -Uri "$Base/api/v1/auth/register" -Method Post `
    -ContentType "application/json" `
    -Body (@{ name = "Demo Errors"; email = $email; password = $password; workspace = "Demo Errors" } | ConvertTo-Json) `
    -SessionVariable session
$csrf = (Invoke-RestMethod -Uri "$Base/api/v1/auth/me" -Method Get -WebSession $session).csrf

function Show-Problem {
    param($ErrorRecord)
    $status = $null
    $message = $ErrorRecord.Exception.Message
    if ($ErrorRecord.ErrorDetails -and $ErrorRecord.ErrorDetails.Message) {
        $raw = $ErrorRecord.ErrorDetails.Message
        try {
            $parsed = $raw | ConvertFrom-Json
            $message = $parsed.message
            $status = $parsed.code
        } catch {
            $message = $raw
        }
    }
    if (-not $status -and $ErrorRecord.Exception.Response) {
        $status = [int]$ErrorRecord.Exception.Response.StatusCode
    }
    return [pscustomobject]@{ Status = $status; Message = $message }
}

function Send-File {
    param([string]$Kind, [string]$Path)
    try {
        $result = Invoke-RestMethod -Uri "$Base/api/v1/umkm/imports/files" -Method Post `
            -Form @{ kind = $Kind; file = Get-Item -LiteralPath $Path } `
            -WebSession $session -Headers @{ "X-CSRF-Token" = $csrf }
        return [pscustomobject]@{ Status = 200; Message = "OK count=$($result.count) total=$($result.total)" }
    } catch {
        return Show-Problem $_
    }
}

$cases = @()
Get-ChildItem -File (Join-Path $Root "error-cases") | Sort-Object Name | ForEach-Object {
    $kind = if ($_.Name -like "17-*") { "receipt" } else { "sale" }
    $cases += [pscustomobject]@{ File = $_.Name; Kind = $kind; Scenario = "isi/ekstensi salah" }
}

# Ekstensi diganti supaya lolos pemilih file, isinya tetap teks.
$fakeCsv = Join-Path $env:TEMP ("teks-palsu-" + $stamp + ".csv")
Copy-Item -LiteralPath (Join-Path $Root "error-cases\01-teks-biasa.txt") -Destination $fakeCsv
$cases += [pscustomobject]@{ File = "(salinan) 01-teks-biasa sebagai .csv"; Kind = "sale"; Scenario = "ekstensi dipalsukan" }

$rows = @()
foreach ($case in $cases) {
    $path = if ($case.File -like "(salinan)*") { $fakeCsv } else { Join-Path $Root (Join-Path "error-cases" $case.File) }
    $result = Send-File -Kind $case.Kind -Path $path
    $rows += [pscustomobject]@{
        File = $case.File; Scenario = $case.Scenario; Kind = $case.Kind;
        Status = $result.Status; Message = $result.Message
    }
}

function Send-Json {
    param([string]$Path, $Body, [switch]$WithoutCsrf)
    $headers = @{}
    if (-not $WithoutCsrf) { $headers["X-CSRF-Token"] = $csrf }
    try {
        $result = Invoke-RestMethod -Uri "$Base$Path" -Method Post -ContentType "application/json" `
            -Body ($Body | ConvertTo-Json -Compress) -WebSession $session -Headers $headers
        return [pscustomobject]@{ Status = 200; Message = "OK $($result | ConvertTo-Json -Compress)" }
    } catch {
        return Show-Problem $_
    }
}

$commitEmpty = Send-Json -Path "/api/v1/umkm/imports" -Body @{ kind = "sale"; csv = "" }
$commitBadKind = Send-Json -Path "/api/v1/umkm/imports" -Body @{ kind = "pengeluaran"; csv = "tanggal,nama,nominal`n2026-11-03,Budi,1000" }

$rows += [pscustomobject]@{ File = "commit csv kosong"; Scenario = "commit tanpa isi"; Kind = "sale"; Status = $commitEmpty.Status; Message = $commitEmpty.Message }
$rows += [pscustomobject]@{ File = "commit kind pengeluaran"; Scenario = "kind tidak dikenal"; Kind = "pengeluaran"; Status = $commitBadKind.Status; Message = $commitBadKind.Message }

$rows | Format-Table -AutoSize -Wrap

$clearBody = @{ password = $password; confirm = "HAPUS" } | ConvertTo-Json
try {
    $null = Invoke-RestMethod -Uri "$Base/api/v1/umkm/data" -Method Delete -ContentType "application/json" `
        -Body $clearBody -WebSession $session -Headers @{ "X-CSRF-Token" = $csrf }
} catch {
    Write-Warning "gagal membersihkan data: $($_.Exception.Message)"
}
Write-Host ""
Write-Host "akun probe: $email / $password"
