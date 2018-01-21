set PROJECT_DIR=..
set SRC_DIR=%PROJECT_DIR%\sources
set TOOLS_DIR=%PROJECT_DIR%\tools

@echo %TOOLS_DIR%

%TOOLS_DIR%\proto\protoc --java_out=%TOOLS_DIR%\ProtoTestServerSicBo\src/main\java -I %SRC_DIR%\common\proto %SRC_DIR%\common\proto\msg.proto

copy /y "%SRC_DIR%\common\proto\msg.proto" "%SRC_DIR%/h5/bin/h5/res/"

%TOOLS_DIR%\proto\protoc -I %SRC_DIR%\common\proto --gotemplate_out=template_dir=%SRC_DIR%\common\proto\client,debug=true,all=true:%SRC_DIR%\common\proto\client %SRC_DIR%\common\proto\msg.proto


%TOOLS_DIR%\proto\mapprotoid --f=%SRC_DIR%\common\proto\client\as_template_dic.txt --t=%SRC_DIR%\h5\modules\core\logic\nets\MessageIDs.as --data=%SRC_DIR%\common\proto\client\arr.js --j=%SRC_DIR%\common\proto\mapids.js
%TOOLS_DIR%\proto\mapprotoid --f=%SRC_DIR%\common\proto\client\java_template_dic.txt --t=%TOOLS_DIR%\ProtoTestServerSicBo\src\main\java\msg\MessageIDs.java --data=%SRC_DIR%\common\proto\client\arr.js --j=%SRC_DIR%\common\proto\mapids.js


%TOOLS_DIR%\proto\mt --data=%SRC_DIR%/common/proto/client/arr.js --data2=%SRC_DIR%/common/proto/mapids.js --f=%SRC_DIR%/common/proto/client/one_proto.as.txt --t=%SRC_DIR%\h5\modules\core\logic\messages\
%TOOLS_DIR%\proto\mt --f=%SRC_DIR%/common/proto/client/handle_proto.java.txt --t=%TOOLS_DIR%/ProtoTestServerSicBo/src/main/java/handler/ --ext=Handler.java --data=%SRC_DIR%/common/proto/client/arr.js --data2=%SRC_DIR%/common/proto/mapids.js
%TOOLS_DIR%\proto\mt --f=%SRC_DIR%/common/proto/client/handle_proto.as.txt --t=%SRC_DIR%/h5/modules/core/logic/handlers/ --ext=Handler.as --data=%SRC_DIR%/common/proto/client/arr.js --data2=%SRC_DIR%/common/proto/mapids.js

@echo ----------------------------  
@echo     Build Proto END  
@echo ----------------------------  
