# 定义变量
GO := go
CMD_DIR := cmd  # 主目录
TARGET := $(firstword $(MAKECMDGOALS))  # 获取目标名称（如 tolink/sensor/mqtt）
BINARY_NAME := $(TARGET).exe  # 生成的可执行文件名

# 默认目标：显示帮助信息
.PHONY: help
help:
	@echo "Usage:"
	@echo "  make <dir>      - 编译并运行 cmd/<dir>/ 下的所有 .go 文件"
	@echo "  make <dir>-run - 仅运行 cmd/<dir>/ 下的所有 .go 文件"
	@echo "  make <dir>-build - 仅编译 cmd/<dir>/ 下的所有 .go 文件"
	@echo "  make clean      - 清理所有生成的可执行文件"
	@echo ""
	@echo "示例:"
	@echo "  make sensor     # 编译并运行 cmd/sensor/"
	@echo "  make mqtt-run  # 仅运行 cmd/mqtt/"

# 默认规则：编译并运行指定目录
# $(TARGET):
# 	$(GO) build -o $(BINARY_NAME) ./$(CMD_DIR)/$@/*.go
# 	@echo "运行程序:"
# 	./$(BINARY_NAME)

# 仅运行指定目录
%:
	$(GO) run ./cmd/$*/ *.go

%-run:
	$(GO) run ./cmd/$*/ *.go

# 仅编译指定目录
%-build:
	$(GO) build -o $(BINARY_NAME) ./$(CMD_DIR)/$*/*.go

# 清理所有可执行文件
.PHONY: clean
clean:
	@del /f /q *.exe 2>nul || true
	@echo "已清理所有可执行文件"