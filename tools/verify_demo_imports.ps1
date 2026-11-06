#!/usr/bin/env pwsh
# Verifikasi file demo impor terhadap API EFA.
# Pemakaian: pwsh tools/verify_demo_imports.ps1 -Base https://temukas.madgo.my.id

param(
    [string]$Base = "https://temukas.madgo.my.id",
    [string]$Root = ""
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($Root)) {
    $Root = Join-Path (Split-Path -Parent $PSScriptRoot) (Join-Path "docs" "demo-files")
}

$stamp = Get-Date -Format "yyyyMMddHHmmss"
$email = "demo.verify.$stamp@temukas.test"
$password = "DemoVerify!2026"

$register = @{
    name      = "Demo Verify"
    email     = $email
    password  = $password
    workspace = "Demo Verify $stamp"
} | ConvertTo-Json

$session = $null
$null = Invoke-RestMethod -Uri "$Base/api/v1/auth/register" -Method Post `
    -ContentType "application/json" -Body $register -SessionVariable session

$me = Invoke-RestMethod -Uri "$Base/api/v1/auth/me" -Method Get -WebSession $session
$csrf = $me.csrf
if ([string]::IsNullOrWhiteSpace($csrf)) { throw "CSRF token tidak tersedia dari /auth/me." }

Write-Host "akun: $email"

function Clear-Data {
    $body = @{ password = $password; confirm = "HAPUS" } | ConvertTo-Json
    $null = Invoke-RestMethod -Uri "$Base/api/v1/umkm/data" -Method Delete `
        -ContentType "application/json" -Body $body -WebSession $session `
        -Headers @{ "X-CSRF-Token" = $csrf }
}

function Import-File {
    param([string]$Kind, [string]$Path, [string]$Sheet = "")
    $form = @{ kind = $Kind; file = Get-Item -LiteralPath $Path }
    if ($Sheet -ne "") { $form["sheet"] = $Sheet }
    Invoke-RestMethod -Uri "$Base/api/v1/umkm/imports/files" -Method Post `
        -Form $form -WebSession $session -Headers @{ "X-CSRF-Token" = $csrf }
}

function Commit-File {
    param([string]$Kind, [string]$Csv)
    $body = @{ kind = $Kind; csv = $Csv } | ConvertTo-Json -Compress
    Invoke-RestMethod -Uri "$Base/api/v1/umkm/imports" -Method Post `
        -ContentType "application/json" -Body $body -WebSession $session `
        -Headers @{ "X-CSRF-Token" = $csrf }
}

$scenarios = @(
    @{ Name = "happy-excel"; Sale = "happy-excel/penjualan-happy-33.xlsx"; Receipt = "happy-excel/mutasi-happy-33.xlsx" },
    @{ Name = "happy-hybrid"; Sale = "happy-hybrid/penjualan-happy-33.csv"; Receipt = "happy-hybrid/mutasi-happy-33.xlsx" },
    @{ Name = "edge-01-fee"; Sale = "edge-cases/01-selisih-fee/fee-penjualan.csv"; Receipt = "edge-cases/01-selisih-fee/fee-mutasi.csv" },
    @{ Name = "edge-02-sebagian"; Sale = "edge-cases/02-bayar-sebagian/sebagian-penjualan.csv"; Receipt = "edge-cases/02-bayar-sebagian/sebagian-mutasi.csv" },
    @{ Name = "edge-03-tunai"; Sale = "edge-cases/03-tunai-dan-belum-bayar/tunai-penjualan.csv"; Receipt = "edge-cases/03-tunai-dan-belum-bayar/tunai-mutasi.csv" },
    @{ Name = "edge-04-kembar"; Sale = "edge-cases/04-nominal-kembar/kembar-penjualan.csv"; Receipt = "edge-cases/04-nominal-kembar/kembar-mutasi.csv" },
    @{ Name = "edge-05-nonop"; Sale = "edge-cases/05-non-operasional/non-operasional-penjualan.csv"; Receipt = "edge-cases/05-non-operasional/non-operasional-mutasi.csv" },
    @{ Name = "edge-06-messy"; Sale = "edge-cases/06-format-berantakan/berantakan-penjualan.xlsx"; Receipt = "edge-cases/06-format-berantakan/berantakan-mutasi.xlsx" }
)

$report = @()
try {
$report = foreach ($scenario in $scenarios) {
    Clear-Data
    $salePreview = Import-File -Kind "sale" -Path (Join-Path $Root $scenario.Sale)
    $null = Commit-File -Kind "sale" -Csv $salePreview.csv
    $receiptPreview = Import-File -Kind "receipt" -Path (Join-Path $Root $scenario.Receipt)
    $commit = $null
    if ($receiptPreview.count -gt 0) {
        $commit = Commit-File -Kind "receipt" -Csv $receiptPreview.csv
    }
    $state = Invoke-RestMethod -Uri "$Base/api/v1/umkm/state" -Method Get -WebSession $session
    $exceptions = Invoke-RestMethod -Uri "$Base/api/v1/umkm/exceptions" -Method Get -WebSession $session

    [pscustomobject]@{
        Skenario    = $scenario.Name
        Aturan      = if ($commit) { $commit.auto_matched } else { 0 }
        AI          = if ($commit) { $commit.ai_matched } else { 0 }
        StatusAI    = if ($commit) { $commit.ai_status } else { "-" }
        Penjualan   = @($state.state.entries | Where-Object { $_.kind -eq "sale" }).Count
        Penerimaan  = @($state.state.entries | Where-Object { $_.kind -eq "receipt" }).Count
        Alokasi     = @($state.state.allocations).Count
        Temuan      = ($exceptions | Measure-Object).Count
        Peringatan  = (($receiptPreview.warnings + $salePreview.warnings) -join "; ")
    }
}
}
finally {
    Clear-Data
}

$report | Format-Table -AutoSize -Wrap
Write-Host ""
Write-Host "akun verifikasi: $email / $password"
