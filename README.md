# haravan-cli

CLI và MCP server hợp nhất cho nền tảng thương mại điện tử **Haravan**, viết bằng Go. Một file binary duy nhất, hai cách dùng:

- **CLI** — 70 lệnh con thuộc 8 nhóm để script hoá thao tác và debug từ terminal.
- **MCP server** — mọi lệnh đều được phơi ra dưới dạng tool MCP (qua `stdio` hoặc `HTTP/SSE`) cho **Claude.ai**, **Claude Desktop**, **Claude Code**, **Cursor** và bất kỳ client nào hỗ trợ MCP.

Không cần Node, không cần npm, không runtime đi kèm — chỉ một binary tĩnh ~15 MB.

---

## Mục lục

1. [Cài đặt](#1-cài-đặt)
2. [Cấu hình & xác thực](#2-cấu-hình--xác-thực)
3. [Sử dụng CLI](#3-sử-dụng-cli)
4. [Chạy MCP server](#4-chạy-mcp-server)
5. [Tích hợp Claude Desktop / Claude Code / Cursor](#5-tích-hợp-claude-desktop--claude-code--cursor)
6. [Claude Skill](#6-claude-skill)
7. [Biến môi trường](#7-biến-môi-trường)
8. [Cấu trúc dự án](#8-cấu-trúc-dự-án)
9. [Phát triển](#9-phát-triển)
10. [Giấy phép](#10-giấy-phép)

---

## 1. Cài đặt

Chọn **một** trong ba cách dưới đây.

### a) Cài bằng `go install` (nhanh nhất)

```bash
go install github.com/pluginmd/haravan-cli@latest
```

> Yêu cầu Go ≥ 1.25. Binary sẽ nằm ở `$(go env GOPATH)/bin/haravan-cli`.

### b) Build từ source

```bash
git clone https://github.com/pluginmd/haravan-cli
cd haravan-cli
make install            # cài vào /usr/local/bin/haravan-cli
```

Các target Make hữu ích:

| Lệnh | Tác dụng |
|------|----------|
| `make build` | Build binary tại thư mục hiện tại (có nhúng version) |
| `make install` | Build + copy sang `/usr/local/bin/` |
| `make uninstall` | Gỡ binary đã cài |
| `make test` | Chạy `go test -race ./...` |
| `make clean` | Xoá binary và artefact build |

### c) Chạy bằng Docker

```bash
cp docker/.env.example docker/.env
# Mở docker/.env, điền HARAVAN_ACCESS_TOKEN
docker compose -f docker/docker-compose.yml up -d
```

Container sẽ phơi MCP server qua HTTP/SSE tại cổng `:4567` (đổi bằng biến `HARAVAN_MCP_PORT`).

---

## 2. Cấu hình & xác thực

Có **2 luồng** cấp token. Chọn một trong hai tuỳ kịch bản.

### a) Private-app token (đơn giản nhất)

Phù hợp khi bạn dùng cho 1 shop và đã có sẵn access token.

```bash
export HARAVAN_ACCESS_TOKEN=xxxxxxxxxxxxxxxx
haravan-cli shop get
```

### b) OAuth 2.0 (khuyến nghị cho app multi-shop)

```bash
# 1. Đăng ký App ID / Secret
haravan-cli config set --app-id=YOUR_APP_ID --app-secret=YOUR_APP_SECRET

# 2. Mở browser để đăng nhập, token sẽ được lưu cục bộ
haravan-cli auth login

# 3. Kiểm tra token đã lưu
haravan-cli auth status
```

### Nơi lưu cấu hình

Mặc định toàn bộ config nằm ở `~/.haravan-cli/` (đổi bằng `HARAVAN_CLI_HOME`):

```
~/.haravan-cli/
├── config.json     # api_base, app_id, app_secret
└── tokens.json     # khoá theo app_id, perm 0600, ghi atomic
```

### Thứ tự ưu tiên token

Mỗi API call sẽ tìm token theo thứ tự:

1. Cờ `--token` truyền trực tiếp trên dòng lệnh
2. Biến môi trường `HARAVAN_ACCESS_TOKEN`
3. Token OAuth đã lưu cho `app_id` đang dùng (tự refresh khi hết hạn nếu có refresh token)

---

## 3. Sử dụng CLI

### Cây lệnh tổng quan

```text
haravan-cli
├── auth              login, logout, status
├── config            show, set, path
├── mcp               serve (stdio | HTTP/SSE), tools
├── orders            13 tool: list, get, create, confirm, close, cancel, …
├── products          11 tool: list, get, create, variants, …
├── customers         14 tool: list, search, groups, addresses, …
├── inventory          5 tool: adjustments, adjust_or_set, locations
├── shop               6 tool: shop info, locations, users, shipping_rates
├── content           11 tool: pages, blogs, articles, script_tags
├── webhooks           3 tool: list, subscribe, unsubscribe
└── smart              7 tool: orders_summary, top_products, rfm, …
```

> Mỗi tool **vừa** là lệnh shell **vừa** là MCP tool — phần handler chỉ viết một lần, dùng chung cho cả hai mặt.

### Ví dụ thường dùng

```bash
# Xem thông tin shop
haravan-cli shop get

# Liệt kê đơn hàng hôm nay (auto pagination)
haravan-cli orders list \
  --status=any \
  --created_at_min=2026-04-24T00:00:00Z \
  --fetch_all

# Tạo đơn từ file JSON
haravan-cli orders create --body=@order.json

# Báo cáo doanh thu 30 ngày + so sánh kỳ trước
haravan-cli smart orders_summary

# Phân khúc khách hàng RFM
haravan-cli smart customer_segments --min_orders=1

# Liệt kê toàn bộ tool mà MCP server sẽ phơi ra
haravan-cli mcp tools
```

### Quy ước truyền JSON body

Với các lệnh `create` / `update`:

- Có thể truyền JSON inline: `--body='{"name":"abc"}'`
- Hoặc trỏ tới file: `--body=@path/to/file.json`
- Lớp envelope ngoài (`{"order": …}`, `{"product": …}`, …) **được tự động bọc** nếu bạn truyền vào object thuần.

---

## 4. Chạy MCP server

### Chế độ stdio (mặc định cho MCP client)

```bash
haravan-cli mcp serve
```

### Chế độ HTTP/SSE (truy cập qua mạng)

```bash
haravan-cli mcp serve --http --addr=:4567
```

Endpoint SSE sẽ ở `http://<host>:4567/sse`.

---

## 5. Tích hợp Claude Desktop / Claude Code / Cursor

### Cấu hình chuẩn (khuyến nghị: dùng wrapper Keychain trên macOS)

Để **không** đặt token trực tiếp trong `claude_desktop_config.json`, dùng wrapper đọc token từ macOS Keychain:

```bash
# 1. Lưu token vào Keychain (chỉ lần đầu hoặc khi rotate)
security add-generic-password -U \
  -a "$USER" -s "haravan-cli-token" -w 'YOUR_TOKEN'

# 2. Cài wrapper vào ~/.local/bin (bắt buộc, vì macOS TCC chặn script
#    chạy từ ~/Downloads kể cả khi binary chạy được)
make install-claude-wrapper
```

Sau đó cấu hình Claude Desktop / Claude Code:

```json
{
  "mcpServers": {
    "haravan": {
      "command": "/Users/<bạn>/.local/bin/haravan-mcp",
      "args": ["mcp", "serve"]
    }
  }
}
```

### Cấu hình rút gọn (token để thẳng trong env)

Nếu không dùng wrapper:

```json
{
  "mcpServers": {
    "haravan": {
      "command": "haravan-cli",
      "args": ["mcp", "serve"],
      "env": {
        "HARAVAN_ACCESS_TOKEN": "your-token"
      }
    }
  }
}
```

### Cursor

Đặt cùng cấu trúc trên vào `~/.cursor/mcp.json`.

> **Lưu ý macOS TCC:** đừng để wrapper script trong `~/Downloads/`, `~/Documents/`, … macOS sẽ **âm thầm** không cho Claude Desktop spawn nó. Luôn cài vào `~/.local/bin` hoặc `/usr/local/bin`.

---

## 6. Claude Skill

Thư mục [`claudeskill/haravan-mcp/`](claudeskill/haravan-mcp/) là **lớp suy luận** thiết kế cho Claude.ai. Nó gồm:

- Tài liệu mô tả chi tiết từng tool MCP
- Decision tree: khi nào dùng tool nào
- Benchmark số liệu thương mại điện tử Việt Nam (RFM, AOV, conversion rate, …)

Cách dùng:

- **Claude.ai (web):** upload nguyên thư mục dưới dạng Skill
- **Claude Code:** copy sang `~/.claude/skills/haravan-mcp/`

---

## 7. Biến môi trường

| Biến | Mục đích |
|------|----------|
| `HARAVAN_ACCESS_TOKEN` | Private-app token |
| `HARAVAN_APP_ID` | OAuth client_id |
| `HARAVAN_APP_SECRET` | OAuth client_secret |
| `HARAVAN_API_BASE` | Override URL API chính (mặc định `https://apis.haravan.com`) |
| `HARAVAN_WEBHOOK_BASE` | Override URL webhook API (mặc định `https://webhook.haravan.com`) |
| `HARAVAN_CLI_HOME` | Override thư mục lưu config/token (mặc định `~/.haravan-cli`) |
| `HARAVAN_LOG_LEVEL` | `debug` / `info` / `warn` / `error` / `off` |

---

## 8. Cấu trúc dự án

```
haravan-cli/
├── main.go                # entrypoint
├── cmd/                   # định nghĩa lệnh CLI (cobra)
│   ├── auth/              # auth login | logout | status
│   ├── cfg/               # config show | set | path
│   ├── mcp/               # mcp serve | tools
│   └── root.go            # root command + flag chung
├── internal/
│   ├── auth/              # OAuth flow + lưu token
│   ├── build/             # version/commit/date inject lúc build
│   ├── client/            # Haravan REST client (retry, pagination)
│   ├── cmdutil/           # helper dùng chung cho cmd/
│   ├── config/            # đọc/ghi ~/.haravan-cli/
│   ├── iostreams/         # stdin/stdout/stderr abstraction
│   ├── logger/            # leveled logger
│   └── mcpserver/         # MCP server (stdio + HTTP/SSE)
├── pkg/                   # API package public (có thể import)
├── claudeskill/           # Claude Skill (reasoning layer)
├── docker/                # Dockerfile + docker-compose
├── docs/                  # tài liệu chi tiết
│   ├── system-architecture.md
│   ├── code-standards.md
│   └── deployment-guide.md
├── scripts/
│   └── claude-desktop-wrapper.sh   # wrapper đọc token từ Keychain
├── Makefile
└── README.md              # file này
```

---

## 9. Phát triển

```bash
make build               # build binary với version inject
make test                # go test -race ./...
make vet                 # go vet ./...
make lint                # cần golangci-lint
make release-snapshot    # goreleaser snapshot cục bộ
```

Tài liệu kiến trúc, design decision và hướng dẫn đóng góp xem trong [docs/](docs/):

- [docs/system-architecture.md](docs/system-architecture.md) — kiến trúc hệ thống
- [docs/code-standards.md](docs/code-standards.md) — chuẩn code
- [docs/deployment-guide.md](docs/deployment-guide.md) — hướng dẫn deploy

---

## 10. Giấy phép

MIT — xem [LICENSE](LICENSE).
