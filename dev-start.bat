@echo off
chcp 65001 >nul
echo ========================================
echo   FnTube 本地调试环境
echo ========================================
echo.
echo 启动后端 (Go, port 8080)...
echo.

set "FN_APP_PORT=8080"
set "TRIM_PKGVAR=%CD%\.dev\pkvar"
set "TRIM_APPDEST=%CD%"

if not exist ".dev\pkvar" mkdir ".dev\pkvar"

cmd /c "cd backend && go run . && pause"
