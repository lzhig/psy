call global.bat

@echo %TOOLS_DIR%

%TOOLS_DIR%\protobuf\protobuf-net\protogen.exe -I%SRC_DIR%\common\protobuf\ --csharp_out=%SRC_DIR%\unity\ msg.proto