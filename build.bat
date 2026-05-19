@echo off
:: Build script for Discord Portable - launcher + updater binaries
:: Requires: Go 1.21+ and `go install github.com/akavel/rsrc@latest`
setlocal
set ROOT=%~dp0

set "RSRC=rsrc"
where %RSRC% >nul 2>nul
if errorlevel 1 set "RSRC=%USERPROFILE%\go\bin\rsrc.exe"

if not exist "%ROOT%_src\app.ico" (
    echo [!] Missing "%ROOT%_src\app.ico". Place the Discord icon there.
    exit /b 1
)

echo [1/4] Copying icon into source folders...
copy /Y "%ROOT%_src\app.ico" "%ROOT%_src\cmd\launcher\app.ico" >nul
copy /Y "%ROOT%_src\app.ico" "%ROOT%_src\cmd\updater\app.ico"  >nul

echo [2/4] Generating Windows resources...
"%RSRC%" -manifest "%ROOT%_src\cmd\launcher\app.manifest" -ico "%ROOT%_src\cmd\launcher\app.ico" -arch amd64 -o "%ROOT%_src\cmd\launcher\rsrc_amd64.syso"
if errorlevel 1 goto :err
"%RSRC%" -manifest "%ROOT%_src\cmd\updater\app.manifest" -ico "%ROOT%_src\cmd\updater\app.ico" -arch amd64 -o "%ROOT%_src\cmd\updater\rsrc_amd64.syso"
if errorlevel 1 goto :err

echo [3/4] Building Launch_Discord.exe (silent, GUI subsystem)...
pushd "%ROOT%_src\cmd\launcher"
go build -ldflags "-s -w -H windowsgui" -trimpath -o "%ROOT%Launch_Discord.exe" .
popd
if errorlevel 1 goto :err

echo [4/4] Building Discord_Updater.exe (console)...
pushd "%ROOT%_src\cmd\updater"
go build -ldflags "-s -w" -trimpath -o "%ROOT%Discord_Updater.exe" .
popd
if errorlevel 1 goto :err

echo.
echo Done. Output:
dir /b "%ROOT%Launch_Discord.exe" "%ROOT%Discord_Updater.exe"
exit /b 0

:err
echo.
echo Build failed.
exit /b 1
