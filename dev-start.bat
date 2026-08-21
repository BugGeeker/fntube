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
if not exist ".dev\fntube.exe" (
    echo 首次运行，正在编译后端...
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
)

start "FnTube-Backend" ".dev\fntube.exe"

echo 后端已启动: http://localhost:8080
echo.
echo 启动前端 (Vite, port 5173)...
echo.

cd frontend
start "FnTube-Frontend" npx vite --port 5173
cd ..

echo.
echo ========================================
echo   前端: http://localhost:5173
echo   后端: http://localhost:8080
echo   API:  /api -> localhost:8080
echo ========================================
echo.
echo 按任意键关闭此窗口（不会停止服务）...
pause >nul