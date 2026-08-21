@echo off
chcp 65001 >nul
echo 停止 FnTube 调试服务...
taskkill /FI "WINDOWTITLE eq FnTube-Backend" /F 2>nul
taskkill /FI "WINDOWTITLE eq FnTube-Frontend" /F 2>nul
echo 已停止。
pause
