param(
    # 前端生产构建目录；默认核对同级 yewuCloud 的 DCSY 构建产物。
    [string]$FrontendDist = (Join-Path $PSScriptRoot "..\..\yewuCloud\dist")
)

$ErrorActionPreference = "Stop"

$dicloudRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
if (-not (Test-Path -LiteralPath $FrontendDist)) {
    throw "未找到前端构建目录：$FrontendDist。请先在 yewuCloud 执行 npm run build:dcsy。"
}

# 从真实生产构建中取接口路径，避免把已被 Vite 移除的旧函数误判为在用接口。
$frontendPattern = '(?:api|auth|business|tolink|go/images)/[A-Za-z0-9_?=&${}./:-]+'
$frontendRoutes = Get-ChildItem -LiteralPath $FrontendDist -Recurse -File -Filter *.js |
    Select-String -AllMatches -Pattern $frontendPattern |
    ForEach-Object { $_.Matches.Value } |
    ForEach-Object { ($_ -replace '\?.*$', '') -replace '\$\{.*$', '' } |
    Sort-Object -Unique

# Gin 路由均在三个入口文件中注册；按各自 Group 前缀还原完整路径。
$backendRoutes = @()
$routeFiles = @(
	@{ Path = (Join-Path $dicloudRoot 'cmd\apiserver\routes.go'); Prefix = 'api'; Groups = 'api|router'; RoutePrefixOptional = $true },
    @{ Path = (Join-Path $dicloudRoot 'cmd\auth\routes.go'); Prefix = 'auth'; Groups = 'auth' },
    @{ Path = (Join-Path $dicloudRoot 'cmd\business\routes.go'); Prefix = 'business'; Groups = 'business' },
    @{ Path = (Join-Path $dicloudRoot 'cmd\business\routes.go'); Prefix = 'go/images'; Groups = 'image' },
    @{ Path = (Join-Path $dicloudRoot 'cmd\tolink\routes.go'); Prefix = 'tolink'; Groups = 'tolink|ttdcConfig' }
)

foreach ($routeFile in $routeFiles) {
    $pattern = '(?:' + $routeFile.Groups + ')\.(?:GET|POST|PUT|DELETE|PATCH)\("([^"]+)"'
    Get-Content -LiteralPath $routeFile.Path | ForEach-Object {
        if ($_ -match $pattern) {
			$routePath = $Matches[1]
			# apiserver 中既有 Group 相对路径，也有 router 的完整 /api 路径。
			if ($routeFile.RoutePrefixOptional -and $routePath.StartsWith('/api/')) {
				$backendRoutes += $routePath.TrimStart('/')
			} else {
				$backendRoutes += $routeFile.Prefix + $routePath
			}
        }
    }
}
$backendRoutes = $backendRoutes | Sort-Object -Unique

$missingRoutes = Compare-Object -ReferenceObject $backendRoutes -DifferenceObject $frontendRoutes |
    Where-Object SideIndicator -eq '=>' |
    ForEach-Object InputObject

if ($missingRoutes.Count -gt 0) {
    Write-Error ("DCSY 前端存在未注册的后端接口：`n - " + ($missingRoutes -join "`n - "))
    exit 1
}

Write-Host "DCSY 接口路由核对通过：前端使用 $($frontendRoutes.Count) 条，均已在 dicloud 注册。"
