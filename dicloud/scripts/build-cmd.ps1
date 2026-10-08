[CmdletBinding()]
param(
    [Parameter(Position = 0, ValueFromRemainingArguments = $true)]
    [string[]]$Services
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$startedAt = Get-Date
$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$cmdRoot = Join-Path $repoRoot "cmd"
$outputRoot = Join-Path $repoRoot "output"
$stagingRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("dicloud-cmd-build-" + [guid]::NewGuid().ToString("N"))
$environmentNames = @("GOOS", "GOARCH", "CGO_ENABLED")
$originalEnvironment = @{}
$locationChanged = $false
$exitCode = 0

function Get-AvailableServices {
    Get-ChildItem -LiteralPath $cmdRoot -Directory |
        Where-Object {
            @(Get-ChildItem -LiteralPath $_.FullName -Filter "*.go" -File |
                Where-Object { $_.Name -notlike "*_test.go" }).Count -gt 0
        } |
        Sort-Object Name |
        Select-Object -ExpandProperty Name
}

function Resolve-RequestedServices {
    param(
        [string[]]$AvailableServices,
        [string[]]$RequestedServices
    )

    $availableMap = @{}
    foreach ($service in $AvailableServices) {
        $availableMap[$service.ToLowerInvariant()] = $service
    }

    $normalizedRequests = @(
        foreach ($value in $RequestedServices) {
            foreach ($name in ($value -split ",")) {
                $trimmedName = $name.Trim()
                if ($trimmedName.Length -gt 0) {
                    $trimmedName
                }
            }
        }
    )
    if ($normalizedRequests.Count -eq 0) {
        return @($AvailableServices)
    }

    $resolved = New-Object System.Collections.Generic.List[string]
    $seen = @{}
    $invalid = New-Object System.Collections.Generic.List[string]
    foreach ($requestedService in $normalizedRequests) {
        $key = $requestedService.ToLowerInvariant()
        if (-not $availableMap.ContainsKey($key)) {
            $invalid.Add($requestedService)
            continue
        }
        if (-not $seen.ContainsKey($key)) {
            $resolved.Add($availableMap[$key])
            $seen[$key] = $true
        }
    }

    if ($invalid.Count -gt 0) {
        throw "Unknown service(s): $($invalid -join ', '). Available services: $($AvailableServices -join ', ')"
    }
    return @($resolved)
}

function Assert-MainPackages {
    param([string[]]$ServiceNames)

    foreach ($service in $ServiceNames) {
        $packagePath = "./cmd/$service"
        $packageName = (& go list -f "{{.Name}}" $packagePath 2>&1 | Out-String).Trim()
        if ($LASTEXITCODE -ne 0) {
            throw "Unable to inspect service '$service': $packageName"
        }
        if ($packageName -ne "main") {
            throw "cmd/$service is not a main package (package=$packageName)"
        }
    }
}

function Remove-StagingDirectory {
    if (-not (Test-Path -LiteralPath $stagingRoot)) {
        return
    }

    $tempRoot = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath()).TrimEnd('\', '/')
    $resolvedStagingRoot = [System.IO.Path]::GetFullPath($stagingRoot)
    $expectedPrefix = $tempRoot + [System.IO.Path]::DirectorySeparatorChar + "dicloud-cmd-build-"
    if (-not $resolvedStagingRoot.StartsWith($expectedPrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to remove unexpected staging directory: $resolvedStagingRoot"
    }
    Remove-Item -LiteralPath $resolvedStagingRoot -Recurse -Force
}

try {
    foreach ($name in $environmentNames) {
        $originalEnvironment[$name] = [System.Environment]::GetEnvironmentVariable($name, "Process")
    }
    [System.Environment]::SetEnvironmentVariable("GOOS", "linux", "Process")
    [System.Environment]::SetEnvironmentVariable("GOARCH", "amd64", "Process")
    [System.Environment]::SetEnvironmentVariable("CGO_ENABLED", "0", "Process")

    if (-not (Test-Path -LiteralPath $cmdRoot -PathType Container)) {
        throw "cmd directory does not exist: $cmdRoot"
    }
    if ($null -eq (Get-Command go -ErrorAction SilentlyContinue)) {
        throw "Go executable was not found in PATH"
    }

    Push-Location $repoRoot
    $locationChanged = $true

    $availableServices = @(Get-AvailableServices)
    if ($availableServices.Count -eq 0) {
        throw "No services were found under $cmdRoot"
    }
    $selectedServices = @(Resolve-RequestedServices -AvailableServices $availableServices -RequestedServices $Services)
    Assert-MainPackages -ServiceNames $selectedServices

    Write-Host "Target: linux/amd64 (CGO_ENABLED=0)"
    Write-Host "Services ($($selectedServices.Count)): $($selectedServices -join ', ')"
    Write-Host "Output: $outputRoot"

    New-Item -ItemType Directory -Path $stagingRoot | Out-Null
    foreach ($service in $selectedServices) {
        $packagePath = "./cmd/$service"
        $stagingFile = Join-Path $stagingRoot $service
        Write-Host "[BUILD] $service"
        # Deployment builds do not need VCS stamping; disabling it also prevents Go from invoking Git and touching index.lock.
        & go build -trimpath -buildvcs=false -o $stagingFile $packagePath
        if ($LASTEXITCODE -ne 0) {
            throw "Build failed for service '$service'"
        }
    }

    New-Item -ItemType Directory -Path $outputRoot -Force | Out-Null
    foreach ($service in $selectedServices) {
        $stagingFile = Join-Path $stagingRoot $service
        $outputFile = Join-Path $outputRoot $service
        Move-Item -LiteralPath $stagingFile -Destination $outputFile -Force
        $file = Get-Item -LiteralPath $outputFile
        Write-Host ("[OUTPUT] {0} ({1:N2} MB)" -f $file.FullName, ($file.Length / 1MB))
    }

    $elapsed = (Get-Date) - $startedAt
    Write-Host ("Build completed: {0} service(s), elapsed {1:hh\:mm\:ss}" -f $selectedServices.Count, $elapsed)
}
catch {
    Write-Host "[ERROR] $($_.Exception.Message)" -ForegroundColor Red
    $exitCode = 1
}
finally {
    try {
        Remove-StagingDirectory
    }
    catch {
        Write-Host "[WARN] Failed to clean staging directory: $($_.Exception.Message)" -ForegroundColor Yellow
        if ($exitCode -eq 0) {
            $exitCode = 1
        }
    }

    if ($locationChanged) {
        Pop-Location
    }
    foreach ($name in $environmentNames) {
        [System.Environment]::SetEnvironmentVariable($name, $originalEnvironment[$name], "Process")
    }
}

exit $exitCode
