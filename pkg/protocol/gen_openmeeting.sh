# Copyright © 2023 OpenIM. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#PROTO_NAMES=(
#    "admin"
#    "user"
#    "meeting"
#)
#
#for name in "${PROTO_NAMES[@]}"; do
#  protoc --go_out=plugins=grpc:./openmeeting/${name} --go_opt=module=github.com/openimsdk/protocol/openmeeting/${name} openmeeting/${name}/${name}.proto
#  if [ $? -ne 0 ]; then
#      echo "error processing ${name}.proto"
#      exit $?
#  fi
#done
#
#if [ "$(uname -s)" == "Darwin" ]; then
#    find . -type f -name '*.pb.go' -exec sed -i '' 's/,omitempty"`/\"\`/g' {} +
#else
#    find . -type f -name '*.pb.go' -exec sed -i 's/,omitempty"`/\"\`/g' {} +
#fi

@echo off
setlocal

rem Define array elements
set "PROTO_NAMES=meeting"

rem Loop through each element in the array
for %%i in (%PROTO_NAMES%) do (
    protoc --go_out=plugins=grpc:./%%i --go_opt=module=github.com/openimsdk/protocol/openmeeting/%%i %%i/%%i.proto
    if ERRORLEVEL 1 (
        echo error processing %%i.proto
        exit /b %ERRORLEVEL%
    )
)

rem Replace "omitempty" in *.pb.go files with UTF-8 encoding
for /r %%f in (*.pb.go) do (
    powershell -Command "(Get-Content -Path '%%f' -Encoding UTF8) -replace ',omitempty\"`"', '\"`"' | Set-Content -Path '%%f' -Encoding UTF8"
)


endlocal