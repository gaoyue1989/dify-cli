# dify-cli

Dify 沙盒环境中的工具调用代理。负责在沙盒 VM 内接收工具调用请求并将其转发到 Dify API Server。

## 功能

| 命令 | 说明 |
|------|------|
| `dify init` | 读取 `.dify_cli.json`，为每个工具创建 `tool_name_uuid` 符号链接 |
| `dify list` | 列出所有可用工具引用 |
| `dify env` | 显示当前环境配置 |
| `dify help <tool>` | 通过 API 获取工具参数信息 |
| `dify execute <ref>` | 直接调用指定工具 |
| 符号链接 `./tool_name` | 通过链接名自动识别并调用对应工具 |

## 编译

### Docker 编译（推荐）

```bash
docker run --rm -v "$(pwd)":/app -w /app -e GOPROXY=https://goproxy.cn,direct \
  golang:1.22-alpine sh -c \
  "CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o dify-cli-linux-amd64 ."
```

其他平台：

```bash
# Linux ARM64
... GOOS=linux GOARCH=arm64 ... -o dify-cli-linux-arm64 .

# macOS AMD64
... GOOS=darwin GOARCH=amd64 ... -o dify-cli-darwin-amd64 .

# macOS ARM64
... GOOS=darwin GOARCH=arm64 ... -o dify-cli-darwin-arm64 .
```

### 本地编译

要求 Go 1.22+：

```bash
go mod tidy
CGO_ENABLED=0 go build -ldflags="-s -w" -o dify-cli .
```

## 配置

在沙盒工作目录创建 `.dify_cli.json`：

```json
{
  "env": {
    "files_url": "http://api:5001",
    "cli_api_url": "http://api:5001/cli/api",
    "cli_api_session_id": "session-uuid",
    "cli_api_secret": "random-secret"
  },
  "tool_references": [
    {
      "id": "abc12345-6789-0abc-def0-123456789abc",
      "tool_type": "mcp",
      "tool_name": "search_docs",
      "tool_provider": "mcp_server_1",
      "credential_id": "cred-001",
      "default_value": { "query": "" }
    }
  ]
}
```

也可通过环境变量指定配置路径：

```bash
DIFY_CLI_CONFIG=/custom/path/.dify_cli.json dify init
```

## 使用

```bash
# 初始化符号链接
dify init

# 调用工具（通过符号链接）
./search_docs_abc12345 --query "how to use dify"

# 查看工具帮助
./search_docs_abc12345 --help

# 通过 execute 命令调用
dify execute web_reader_def45678 --url "https://example.com"

# 上传文件作为参数
./some_tool --file "@path/to/file.txt"
```

## 架构

```
沙盒 VM 内代码调用工具
    ↓
dify-cli (Go binary)
    ├── 解析符号链接名 → 查找 ToolReference
    ├── 解析命令行参数
    ├── HMAC-SHA256 签名 HTTP 请求
    └── POST /cli/api/invoke/tool
    ↓
Dify API Server (Python)
    ├── 验证 session + ToolAccessPolicy 白名单
    ├── 调用 MCP / 内置 / 工作流 工具
    └── 长度前缀分块协议流式响应
    ↓
dify-cli 解析响应 → 输出
    ├── text   → stdout
    ├── image  → stderr [image] URL
    ├── file   → stderr [file] name
    ├── blob   → stdout + stderr [blob] mime_type
    ├── json   → stdout 格式化
    └── ...
```

## Dify 集成

dify-cli 在 Dify 沙盒初始化流程中的位置：

```
Workflow 执行
    → AppAssetAttrsInitializer (同步)
    → AppAssetsInitializer (同步)
    → SkillInitializer (同步) → 加载 SkillBundle (含 MCP 工具依赖)
    → DifyCliInitializer (异步)
        ├── 上传 dify-cli 到沙盒 VM
        ├── 创建 CliApiSession (Redis + ToolAccessPolicy)
        ├── 生成 .dify_cli.json 配置
        └── 执行 dify init → 创建符号链接
            ↓
        沙盒标记 ready
```

## 测试

```bash
go test -v .
```

测试覆盖：配置加载、参数解析、响应处理、MIME 检测、HMAC 签名、CLI 集成 共 44 个用例。

## 项目结构

```
dify-cli/
├── main.go
├── command/
│   └── command.go      # CLI 命令
├── config/
│   └── config.go       # 配置加载
├── tool/
│   └── tool.go         # 工具调用核心
├── types/
│   └── types.go        # 类型定义
├── testdata/           # 测试配置
├── dify_cli_test.go    # 单元测试
├── VERIFICATION.md     # 验证文档
└── Makefile
```

## License

Apache-2.0
