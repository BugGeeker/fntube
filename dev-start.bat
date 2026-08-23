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
@REM if not exist ".dev\fntube.exe" (
@REM echo 首次运行，正在编译后端...
cd backend
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -o ..\.dev\fntube.exe .
cd ..
if errorlevel 1 (
    echo 编译失败！
    pause
    exit /b 1
)
echo 编译完成。
@REM )

start "FnTube-Backend" ".dev\fntube.exe"
