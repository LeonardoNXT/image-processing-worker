$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
$environmentPath = Join-Path $projectRoot ".env"

if (-not (Test-Path -LiteralPath $environmentPath)) {
    throw "Arquivo .env não encontrado em $projectRoot"
}

Get-Content -LiteralPath $environmentPath | ForEach-Object {
    if ($_ -match '^\s*([^#][^=]*)=(.*)$') {
        $name = $matches[1].Trim()
        $value = $matches[2].Trim()
        if ($value.Length -ge 2 -and (($value.StartsWith('"') -and $value.EndsWith('"')) -or ($value.StartsWith("'") -and $value.EndsWith("'")))) {
            $value = $value.Substring(1, $value.Length - 2)
        }
        Set-Item -Path "Env:$name" -Value $value
    }
}

Push-Location $projectRoot
try {
    & go run ./cmd/worker
}
finally {
    Pop-Location
}
