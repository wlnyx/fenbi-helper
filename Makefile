# 粉笔复盘工作台构建/测试

GO ?= go
NPM ?= npm
BINARY := fenbi-workbench

.PHONY: all build frontend go-build test lint vet clean run

all: build

## 完整构建：前端 + 拷贝 embed + Go 编译
build: frontend go-build

## 前端构建（Vite）
frontend:
	cd frontend && $(NPM) run build
	rm -rf internal/web/dist
	cp -r frontend/dist internal/web/dist

## Go 编译（单二进制）
go-build:
	$(GO) build -trimpath -ldflags "-s -w" -o $(BINARY) ./cmd/server

## 运行测试
test:
	$(GO) test ./...

## 静态检查
vet:
	$(GO) vet ./...

## 全部质量检查
lint: vet test

## 清理构建产物
clean:
	rm -f $(BINARY)
	rm -rf frontend/dist internal/web/dist

## 本地运行（默认端口 3000）
run: build
	./$(BINARY) -port 3000
