@echo off
setlocal

cd /d "%~dp0.."

echo ========================================
echo   FnTube FPK Build Script
echo ========================================
echo.

if not exist "app\server" mkdir "app\server"

echo [1/3] Building Linux backend...
set "CGO_ENABLED=0"
set "GOOS=linux"
set "GOARCH=amd64"
pushd backend
go build -o "..\app\server\fntube" .
if errorlevel 1 goto :error_backend
popd

echo.
echo [2/3] Building frontend...
pushd frontend
call pnpm run build
if errorlevel 1 goto :error_frontend
popd

echo.
echo [3/3] Packing FPK...
fnpack build --directory "%CD%"
if errorlevel 1 goto :error_pack

echo.
echo Build completed: %CD%\FnTube.fpk
exit /b 0

:error_backend
popd
echo Backend build failed.
exit /b 1

:error_frontend
popd
echo Frontend build failed.
exit /b 1

:error_pack
echo FPK packaging failed.
exit /b 1
