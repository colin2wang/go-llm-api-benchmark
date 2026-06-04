@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

REM =====================================================
REM  go-llm-api-benchmark Cross-Compile Script
REM  Output: bin\llm-api-benchmark_win_x64.exe
REM          bin\llm-api-benchmark_linux_x64
REM          bin\config.yaml.example
REM          bin\test-cases\*
REM =====================================================

set APP_NAME=llm-api-benchmark
set BIN_DIR=bin

if not exist %BIN_DIR% mkdir %BIN_DIR%

echo ========================================
echo  Cross-Compile %APP_NAME%
echo ========================================
echo.

REM ---- Windows x64 ----
echo [1/2] Building Windows x64 ...
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w" -o "%BIN_DIR%/%APP_NAME%_win_x64.exe" .
if %ERRORLEVEL% neq 0 (
    echo [FAIL] Windows x64 build failed
    exit /b 1
)
echo   [OK] %BIN_DIR%/%APP_NAME%_win_x64.exe
echo.

REM ---- Linux x64 ----
echo [2/2] Building Linux x64 ...
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o "%BIN_DIR%/%APP_NAME%_linux_x64" .
if %ERRORLEVEL% neq 0 (
    echo [FAIL] Linux x64 build failed
    exit /b 1
)
echo   [OK] %BIN_DIR%/%APP_NAME%_linux_x64
echo.

REM ---- Copy config template ----
echo [3/4] Copying config.yaml.example ...
copy config.yaml.example "%BIN_DIR%\config.yaml.example" > nul
echo   [OK] %BIN_DIR%\config.yaml.example
echo.

REM ---- Copy test-cases ----
echo [4/4] Copying test-cases ...
if exist "%BIN_DIR%\test-cases" rmdir /s /q "%BIN_DIR%\test-cases"
xcopy /E /I /Q test-cases "%BIN_DIR%\test-cases" > nul
echo   [OK] %BIN_DIR%\test-cases\
echo.

echo ========================================
echo  [DONE] All builds completed
echo ========================================
echo.
dir "%BIN_DIR%" /b
echo.
dir "%BIN_DIR%\test-cases" /b
