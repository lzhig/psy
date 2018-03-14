call global.bat

@echo %TOOLS_DIR%

rem %TOOLS_DIR%\protobuf\protobuf-net\protogen.exe -I%SRC_DIR%\common\protobuf\ --csharp_out=%SRC_DIR%\unity\ msg.proto
%TOOLS_DIR%\protobuf\protoc -I%SRC_DIR%\common\protobuf\ --csharp_out=%SRC_DIR%\unity\  msg.proto