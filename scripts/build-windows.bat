@echo off
REM Build script for SourceControl CLI on Windows

setlocal enabledelayedexpansion

echo.
echo ========================================
echo   Building SourceControl for Windows
echo ========================================
echo.

REM Set build directory
set BUILD_DIR=%~dp0..\sourcecontrol
set OUTPUT_DIR=%USERPROFILE%\.sourcecontrol\bin
set OUTPUT_FILE=%OUTPUT_DIR%\srcc.exe

REM Get version info
for /f "tokens=*" %%i in ('git rev-parse --short HEAD 2^>nul') do set COMMIT_SHA=%%i
if "%COMMIT_SHA%"=="" set COMMIT_SHA=unknown

for /f "tokens=*" %%i in ('powershell -Command "Get-Date -Format 'yyyy-MM-dd_HH:mm:ss'"') do set BUILD_TIME=%%i

set VERSION=0.1.0

echo [1/5] Cleaning previous builds...
if exist "%OUTPUT_DIR%\srcc.exe" del /q "%OUTPUT_DIR%\srcc.exe"

echo [2/5] Creating output directory...
if not exist "%OUTPUT_DIR%" mkdir "%OUTPUT_DIR%"

echo [3/5] Building executable...
cd /d "%BUILD_DIR%"
go build -ldflags="-X main.Version=%VERSION% -X main.BuildTime=%BUILD_TIME% -X main.CommitSHA=%COMMIT_SHA%" -o "%OUTPUT_FILE%" ./cmd/sourcecontrol

if errorlevel 1 (
    echo.
    echo ❌ Build failed!
    exit /b 1
)

echo [4/5] Verifying build...
if not exist "%OUTPUT_FILE%" (
    echo.
    echo ❌ Executable not found!
    exit /b 1
)

echo [5/5] Setting up environment...
REM Check if directory is already in PATH
echo %PATH% | find /i "%OUTPUT_DIR%" >nul
if errorlevel 1 (
    echo.
    echo Adding to PATH...
    setx PATH "%PATH%;%OUTPUT_DIR%"
    echo.
    echo  PATH updated! Please restart your terminal for changes to take effect.
) else (
    echo PATH already configured.
)

echo.
echo ========================================
echo   Build completed successfully!
echo ========================================
echo.
echo Executable location: %OUTPUT_FILE%
echo.
echo You can now run:
echo   srcc --help
echo.
echo If 'srcc' is not recognized, restart your terminal.
echo.

endlocal
