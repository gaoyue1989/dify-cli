# dify-cli 验证文档

## 一、概述

`dify-cli` 是 Dify 沙盒环境中的工具调用代理，负责在沙盒 VM 内接收工具调用请求并将其转发到 Dify API Server。

本文档记录完整的验证方案和测试结果。

---

## 二、二进制验证

### 2.1 完整性校验

```bash
# 校验 SHA256
sha256sum /app/api/bin/dify-cli-linux-amd64
# e70449e40adcae673947a5ebc86dfb7e06f04081972872f7ea93fb22c3100aa5

# 文件类型
file /app/api/bin/dify-cli-linux-amd64
# ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, stripped
```

### 2.2 多平台产物

| 文件 | 大小 | 目标 |
|------|------|------|
| `dify-cli-linux-amd64` | 4.9M | Linux x86_64 |
| `dify-cli-linux-arm64` | 4.8M | Linux ARM64 |
| `dify-cli-darwin-amd64` | 5.1M | macOS x86_64 |
| `dify-cli-darwin-arm64` | 4.8M | macOS ARM64 |

---

## 三、功能验证

### 3.1 CLI 命令列表

| 命令 | 功能 | 验证方法 |
|------|------|----------|
| `dify --help` | 显示帮助 | 输出包含 "Dify CLI for sandbox tool invocation" |
| `dify init` | 创建工具符号链接 | 读取 `.dify_cli.json`，为每个 ToolReference 创建 `tool_name_uuid` 链接 |
| `dify list` | 列出可用工具 | 输出 "Available tools:" 及所有工具 |
| `dify env` | 显示环境配置 | 输出 CLI API URL / Session ID / Tool count |
| `dify help <tool>` | 显示工具帮助 | 通过 API 获取工具参数并格式化输出 |
| `dify execute <ref>` | 直接执行工具 | 等价于符号链接调用 |
| 符号链接 `./tool_name` | 通过链接名识别并调用工具 | 解析链接名 → 查配置 → 调 API |

### 3.2 配置文件格式

`.dify_cli.json` 是一个 JSON 文件，由 Dify Python 端 `DifyCliInitializer` 自动生成：

```json
{
  "env": {
    "files_url": "http://api:5001",
    "cli_api_url": "http://api:5001/cli/api",
    "cli_api_session_id": "uuid-session-id",
    "cli_api_secret": "urlsafe-random-secret"
  },
  "tool_references": [
    {
      "id": "abc12345-6789-0abc-def0-123456789abc",
      "tool_type": "mcp",
      "tool_name": "search_docs",
      "tool_provider": "mcp_server_1",
      "credential_id": "cred-001",
      "default_value": {"query": ""}
    }
  ]
}
```

配置可通过环境变量指定路径：
```bash
DIFY_CLI_CONFIG=/custom/path/.dify_cli.json dify init
```

---

## 四、API 交互验证

### 4.1 API 端点

dify-cli 通过以下端点与 Dify API Server 通信：

| 端点 | 方法 | 用途 |
|------|------|------|
| `/cli/api/invoke/tool` | POST | 调用工具 |
| `/cli/api/invoke/llm` | POST | 调用 LLM 模型 |
| `/cli/api/invoke/app` | POST | 调用子应用 |
| `/cli/api/upload/file/request` | POST | 获取文件上传签名 URL |
| `/cli/api/fetch/tools/batch` | POST | 批量获取工具元数据 |

### 4.2 认证机制

每个请求携带三个 Header：

```http
X-Cli-Api-Session-Id: {session_id}
X-Cli-Api-Timestamp: {unix_timestamp}
X-Cli-Api-Signature: sha256={hex_hmac}
```

签名算法：
```
message = "{METHOD}\n{PATH}\n{TIMESTAMP}"
signature = sha256={HMAC-SHA256(secret, message)}
```

### 4.3 响应处理

| 响应类型 | 输出位置 | 格式 |
|---------|---------|------|
| `text` | stdout | 原始文本 |
| `image` / `image_link` | stderr | `[image] URL` |
| `file` | stderr | `[file] filename (URL)` |
| `blob` | stdout + stderr | `[blob] mime_type=...` + 二进制数据 |
| `blob_chunk` | stdout | 分块二进制数据 |
| `json` | stdout | 格式化 JSON |
| `link` | stdout | URL |
| `log` | stderr | `[log] id=... label=... status=...` |
| `variable` | stderr | `[variable] name = value` |
| `variable:stream` | stderr | `[variable:stream] name = value` |
| `binary_link` | stderr | `[binary_link] URL` |
| `retriever_resources` | stderr | 资源上下文 |

### 4.4 响应协议（长度前缀分块）

```
+-------+--------+-----------+-----------+--------+------+
| Magic | Resvd  | HdrLen    | DataLen   | Resvd  | Data |
| 0x0F  | 0x00   | 0x000A    | 4 bytes   | 6 zero | JSON |
+-------+--------+-----------+-----------+--------+------+
   1B      1B       2B LE       4B LE      6B       var
```

---

## 五、测试用例

### 5.1 单元测试（Go）

在 `dify-cli` 源码目录运行：

```bash
go test -v ./...
```

**测试明细（44 用例）：**

| 类别 | 用例数 | 覆盖内容 |
|------|--------|---------|
| Config Load | 3 | `valid.json` 加载、`empty_tools.json` 加载、文件不存在 |
| FindToolReference | 2 | 正常查找、未找到报错 |
| GetReferenceSymlinkName | 1 | 命名格式 `{name}_{id[:8]}` |
| GetSelfPath | 1 | 路径非空 |
| ParseArgs basic | 1 | `--key value` 格式 |
| ParseArgs equals | 1 | `--key=value` 格式 |
| ParseArgs defaults | 1 | 默认值合并 |
| ParseArgs override | 1 | 命令行覆盖默认值 |
| ParseArgs empty | 1 | 空参数数组 |
| ParseArgs ignore | 1 | 忽略非 flag 参数 |
| ParseArgs mixed | 1 | 混合 `=` 和空格格式 |
| Handle tool messages | 7 | TEXT/IMAGE/VARIABLE/BLOB/LOG/JSON/UNKNOWN |
| MIME detection | 6 | txt/json/png/pdf/unknown/double-ext |
| Sign request | 4 | 基本/一致性/相同输入/不同密钥 |
| CLI help | 1 | `--help` 输出 |
| CLI unknown | 1 | 未知命令报错 |
| CLI empty | 1 | 无参数显示帮助 |
| CLI no config | 1 | 无配置文件报错 |
| CLI init | 1 | 创建 4 个符号链接 |
| CLI init empty | 1 | 空工具列表提示 |
| CLI list | 1 | 列出 4 个工具 |
| CLI env | 1 | 显示环境变量 |
| CLI idempotent | 1 | 重复 init 跳过 |
| CLI symlink | 1 | 符号链接 --help |
| CLI DIFY_CLI_CONFIG | 1 | 环境变量配置路径 |
| CLI execute | 1 | execute 命令调用 |
| CLI help command | 1 | help 命令调用 |

### 5.2 集成测试（Python）

通过 SSH 连接到 agentbox 沙盒环境运行：

```bash
docker exec docker-api-1 python3 /tmp/sandbox_test.py
```

**测试明细（14 用例）：**

| # | 测试 | 验证点 |
|---|------|--------|
| 1 | 上传二进制到沙盒 | 文件完整性 |
| 2 | 沙盒内运行 `--help` | 功能正常 |
| 3 | 创建 `.dify_cli.json` | 配置正确 |
| 4 | `dify init` | 创建 3 个符号链接 |
| 5 | 符号链接验证 | `tool_name_uuid → binary` 链接存在 |
| 6 | `dify list` | 列出所有工具 |
| 7 | `dify env` | 环境变量显示正确 |
| 8 | 符号链接 `--help` | 通过 API 获取工具信息 |
| 9 | 符号链接传参 | 参数正确传递给 API |
| 10 | `dify execute` | 直接执行工具 |
| 11 | `DIFY_CLI_CONFIG` 环境变量 | 自定义配置路径 |
| 12 | 空工具配置 | 正确处理 |
| 13 | 幂等性 | 重复 init 跳过已有 |
| 14 | API 网络连通 | 沙盒内可访问 API Server |

### 5.3 Dify Python 单元测试

```bash
uv run --project api python -m pytest \
  api/tests/unit_tests/core/plugin/test_backwards_invocation_app.py \
  api/tests/unit_tests/core/workflow/nodes/tool/test_tool_node.py \
  api/tests/unit_tests/core/tools/test_plugin_tool.py \
  api/tests/unit_tests/core/plugin/utils/test_chunk_merger.py
```

**24 用例全部通过。**

---

## 六、Dify 环境中的集成路径

### 6.1 沙盒初始化流程

```
Workflow 开始执行
    ↓
app_generator.py 检测 WorkflowFeatures.SANDBOX = enabled
    ↓
SandboxService.create() / create_draft()
    ↓
SandboxBuilder.build()
    │
    ├── [同步] AppAssetAttrsInitializer
    ├── [同步] AppAssetsInitializer / DraftAppAssetsInitializer
    ├── [同步] SkillInitializer → 加载 SkillBundle (含 MCP 等工具依赖)
    │
    └── [异步] DifyCliInitializer.initialize()
        ├── 定位 dify-cli 二进制 (按 OS/Arch)
        ├── 上传到沙盒 VM: /tmp/.dify/{sandbox_id}/bin/dify
        ├── 创建 CliApiSession (Redis + ToolAccessPolicy 白名单)
        ├── 生成 .dify_cli.json 配置
        └── 执行 dify init (创建符号链接)
            ↓
        沙盒标记 ready → 代码节点可执行
```

### 6.2 工具调用流程

```
沙盒内代码调用: ./search_docs_abc12345 --query "test"
    ↓
dify-cli 通过链接名识别工具 → 解析参数
    ↓
POST /cli/api/invoke/tool (HMAC 签名)
    ↓
Dify API 验证 session + ToolAccessPolicy 白名单
    ↓
调用 MCP Server / 内置工具 → 返回结果
    ↓
长度前缀分块协议流式响应
    ↓
dify-cli 解析响应 (text/image/file/blob...) → 输出
```

---

## 七、环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DIFY_CLI_CONFIG` | `CWD/.dify_cli.json` | 配置文件路径 |
| `CLI_API_URL` | `http://localhost:5001` | API Server 地址（在配置中） |
| `FILES_API_URL` | 空 | 文件服务地址（在配置中） |

---

## 八、回归测试命令

```bash
# 1. 验证二进制
/root/dify/api/bin/dify-cli-linux-amd64 --help

# 2. Go 单元测试
cd /root/dify-cli
docker run --rm -v "$(pwd)":/app -w /app -e GOPROXY=https://goproxy.cn,direct \
  golang:1.22-alpine sh -c "go build -ldflags='-s -w' -o /app/dify-cli-linux-amd64 . && go test -v ."

# 3. Dify Python 测试
cd /root/dify
uv run --project api python -m pytest \
  api/tests/unit_tests/core/plugin/test_backwards_invocation_app.py \
  api/tests/unit_tests/core/workflow/nodes/tool/test_tool_node.py \
  api/tests/unit_tests/core/tools/test_plugin_tool.py \
  api/tests/unit_tests/core/plugin/utils/test_chunk_merger.py \
  -o "addopts="

# 4. 沙盒集成测试
docker cp /tmp/sandbox_integration_test.py docker-api-1:/tmp/
docker exec docker-api-1 python3 /tmp/sandbox_integration_test.py

# 5. 重新构建镜像
cd /root/dify/docker
docker compose -f docker-compose.yaml -f docker-compose.override.yaml up -d --build api worker worker_beat
```

---

## 九、测试结果汇总

| 测试层次 | 用例数 | 通过 | 失败 |
|---------|--------|------|------|
| Go 单元测试 | 44 | 44 | 0 |
| Python 单元测试 | 24 | 24 | 0 |
| 沙盒 SSH 集成测试 | 14 | 14 | 0 |
| **合计** | **82** | **82** | **0** |

二进制文件：
- SHA256: `e70449e40adcae673947a5ebc86dfb7e06f04081972872f7ea93fb22c3100aa5`
- 类型: Go 1.22 静态编译，stripped，CGO_ENABLED=0
- 大小: ~5MB
