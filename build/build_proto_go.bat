call global.bat

@echo %TOOLS_DIR%

%TOOLS_DIR%\protobuf\protoc --go_out=%SRC_DIR%\server\src\msg -I %SRC_DIR%\common\protobuf %SRC_DIR%\common\protobuf\msg.proto


REM %TOOLS_DIR%\protobuf\protoc -I %SRC_DIR%\common\proto --gotemplate_out=template_dir=%SRC_DIR%\common\proto\client,debug=true,all=true:%SRC_DIR%\common\proto\server %SRC_DIR%\common\proto\msg.proto


REM %TOOLS_DIR%\protobuf\mapprotoid --f=%SRC_DIR%\common\proto\server\proto_parser.txt --t=%SRC_DIR%\server\src\msg\parser.go --data=%SRC_DIR%\common\proto\server\arr.js --j=%SRC_DIR%\common\proto\mapids.js

@echo ------------------------  
@echo     Build Proto OK   
@echo ------------------------  
@REM @pause