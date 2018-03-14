@echo off

call global.bat

set GOARCH=amd64
set GOOS=linux

cd %SRC_DIR%\server\src
go build -o %VERSIONS_DIR%\test\pusoy_server
cd %BUILD_DIR%

@echo ------------------------  
@echo     Build Server OK   
@echo ------------------------  