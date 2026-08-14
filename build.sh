#!/bin/bash
# 构建脚本：前端构建 + 拷入 embed + Go 编译
set -e
export PATH=/usr/bin:/bin:/usr/local/bin:/home/ediem/.local/go/bin:$PATH
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"

cd "$(dirname "$0")"

echo "=== 1. 前端构建 ==="
(cd frontend && npm run build)

echo "=== 2. 拷贝 dist 到 embed 目录 ==="
rm -rf internal/web/dist
cp -r frontend/dist internal/web/dist

echo "=== 3. Go 编译 ==="
go build -trimpath -ldflags "-s -w" -o fenbi-workbench ./cmd/server
ls -lh fenbi-workbench
echo "构建完成: ./fenbi-workbench"
