@echo off
setlocal
cd /d "%~dp0"
set "TEMUKAS_NODE=node"
where node >nul 2>&1
if errorlevel 1 (
  if exist "%USERPROFILE%\.cache\codex-runtimes\codex-primary-runtime\dependencies\node\bin\node.exe" (
    set "TEMUKAS_NODE=%USERPROFILE%\.cache\codex-runtimes\codex-primary-runtime\dependencies\node\bin\node.exe"
  )
)
"%TEMUKAS_NODE%" frontend\start.mjs
if errorlevel 1 pause
endlocal
