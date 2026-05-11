## 写在前面

---     
![大果](./pic/img.png)

这个项目的90%由ai完成，也是本人第一个有ai参与编写的项目。  
~~下次更新会在看懂代码之后再推新代码~~

~~***这个claude opus4.7的果子就是好，果子大味道足口味正，放在我代码里面就是一种享受，我现在都吃不下那种便宜果子了，没戒的赶紧吃***~~

---  

# ArrowLive 使用文档

基于 ZLMediaKit 的推拉流房间管理系统。用户自助生成推流码并同屏预览，管理员集中监控所有活动直播。

## 目录

- [一、系统概述](#一系统概述)
- [二、前置条件](#二前置条件)
- [三、快速开始](#三快速开始)
- [四、配置文件详解](#四配置文件详解)
- [五、ZLMediaKit 侧配置](#五zlmediakit-侧配置)
- [六、用户端使用](#六用户端使用)
- [七、管理端使用](#七管理端使用)
- [八、OBS 推流配置](#八obs-推流配置)
- [九、HTTP API 参考](#九http-api-参考)
- [十、数据存储与状态机](#十数据存储与状态机)
- [十一、故障排查](#十一故障排查)
- [十二、部署建议](#十二部署建议)

## 一、系统概述

### 1.1 功能范围

- **用户端**：推流者输入房间名，系统返回 RTMP 推流地址与 token，同页内嵌 HTTP-FLV 播放器实时预览推流效果。同一房间名在推流期间禁止他人占用。
- **管理端**：需账号密码登录。提供两种监控视图：
  - 列表模式：左侧大播放器 + 右侧活动房间列表
  - 九宫格模式：按 4 路一组自动切分活动房间，2×2 同屏预览
- **后端**：Go 单进程，SQLite 持久化房间信息，通过 RESTful API 与 WebHook 与 ZLMediaKit 对接。

### 1.2 系统架构

```
┌──────────────┐    HTTP-FLV 播放    ┌──────────────────┐
│ 用户/管理员  │ ◀────────────────── │                  │
│   浏览器     │     RTMP 推流       │   ZLMediaKit     │
│              │ ──────────────────▶ │  (媒体服务器)     │
└──────┬───────┘                     └────────┬─────────┘
       │                                       │
       │ HTTP (Web UI, JSON API)               │ WebHook 回调
       ▼                                       ▼
┌─────────────────────────────────────────────────────────┐
│              ArrowLive Go 服务 (本项目)                   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────┐ │
│  │ /user/*  │  │ /admin/* │  │/webhook/*│  │ SQLite  │ │
│  └──────────┘  └──────────┘  └──────────┘  └─────────┘ │
│       │              │              │                    │
│       └──────────────┴──────────────┘                    │
│                      │                                    │
│                      ▼ RESTful (getMediaList 等)          │
└──────────────────────┼───────────────────────────────────┘
                       │
                       ▼
                 ZLMediaKit /index/api/*
```

### 1.3 端口与 URL 约定（默认配置）

| 角色 | 地址 |
|---|---|
| ArrowLive 服务 | `http://<host>:8080` |
| 用户端入口 | `http://<host>:8080/user/` 或 `http://<host>:8080/` |
| 管理端入口 | `http://<host>:8080/admin/` |
| ZLMediaKit HTTP | `http://<host>:80` |
| ZLMediaKit RTMP | `rtmp://<host>:1935` |

## 二、前置条件

### 2.1 运行时依赖

- **Go 1.26+**（本项目使用 Go 1.26.2）
- **ZLMediaKit** 媒体服务器（已部署并运行，本项目**不**内嵌媒体服务器）
- 操作系统：Windows / Linux / macOS 皆可。Windows 下用 Git Bash 或 PowerShell 均可。

### 2.2 ZLMediaKit 要求

- 已启用 `http` 模块（默认 80 端口）和 `rtmp` 模块（默认 1935 端口）
- 已启用 `hook` 模块并配置回调地址（详见 [§5](#五zlmediakit-侧配置)）
- 能通过 HTTP 访问本 ArrowLive 服务（通常部署在同机或同内网）

### 2.3 构建依赖

仅三个直接依赖，全部纯 Go，无需 cgo：
- `github.com/go-chi/chi/v5` —— HTTP 路由
- `gopkg.in/yaml.v3` —— 配置解析
- `modernc.org/sqlite` —— SQLite 驱动（纯 Go，Windows 下免 cgo）

## 三、快速开始

### 3.1 克隆与构建

```bash
cd C:/Develop/Go/ArrowLiveWebsite

# 拉取依赖（首次）
go mod tidy

# 构建（可选，也可直接 go run）
go build -o arrowlive.exe .
```

### 3.2 准备配置文件

```bash
cp config.yaml.example config.yaml
```

然后用编辑器打开 `config.yaml`，至少修改以下三项：
- `zlmediakit.secret`：必须与 ZLMediaKit `config.ini` 中的 `api.secret` 完全一致
- `zlmediakit.api_base` / `rtmp_push_base` / `http_flv_play_base`：改成 ZLMediaKit 真实地址（浏览器和 OBS 能访问到的）
- `admin.password`：改成一个强口令

### 3.3 启动服务

```bash
# 开发态
go run . -config config.yaml

# 或使用构建产物
./arrowlive.exe -config config.yaml
```

控制台打印 `listening on :8080` 即表示启动成功。

### 3.4 冒烟验证（5 分钟走一遍）

1. 浏览器访问 `http://localhost:8080/`，输入房间名 `test1`，点「生成推流码」
2. 把页面上展示的「推流地址」复制到 OBS
3. 在 OBS 点「开始推流」
4. 刷新页面，2~3 秒内播放器应显示画面，状态徽标变「直播中」
5. 新开标签访问 `/admin/`，用配置文件里的账号密码登录 → 应看到 `test1` 出现在房间列表

## 四、配置文件详解

`config.yaml` 采用标准 YAML 语法。完整示例：

```yaml
server:
  addr: ":8080"          # ArrowLive 监听地址，冒号开头表示所有网卡

zlmediakit:
  api_base: "http://127.0.0.1"      # 调用 ZLM /index/api/* 的 HTTP base
  secret: "035c73f7-bb6b-4889-a715-d9eb2d1925cc"   # 必须与 ZLM 的 api.secret 一致
  rtmp_push_base: "rtmp://127.0.0.1:1935"          # 拼接推流 URL 的 base（OBS 可达）
  http_flv_play_base: "http://127.0.0.1"           # 拼接拉流 URL 的 base（浏览器可达）
  app: "live"                                       # 推拉流的 app 名，一般就用 live

admin:
  username: "admin"      # 管理员账号
  password: "changeme"   # 管理员密码，生产环境务必改掉

database:
  path: "./data.db1"      # SQLite 文件路径，相对路径基于工作目录
```

### 4.1 三个 base 为什么要分开

典型部署里，ZLMediaKit、ArrowLive、浏览器、OBS 的网络位置不一定相同：

| base | 作用方 | 举例 |
|---|---|---|
| `api_base` | ArrowLive 后端调 ZLM | 内网直连，如 `http://127.0.0.1` |
| `rtmp_push_base` | OBS（推流者电脑）→ ZLM | 公网域名，如 `rtmp://live.example.com:1935` |
| `http_flv_play_base` | 观看者浏览器 → ZLM | 公网域名，如 `http://live.example.com` |

分开配置允许后端走内网短链、客户端走公网域名，且方便加 CDN 或反向代理。

### 4.2 必填项与默认值

| 字段 | 必填 | 默认值 |
|---|---|---|
| `zlmediakit.secret` | 是 | — |
| `admin.username` / `admin.password` | 是 | — |
| `server.addr` | 否 | `:8080` |
| `zlmediakit.app` | 否 | `live` |
| `database.path` | 否 | `./data.db` |

启动时若缺少必填项会直接退出并打印错误。

## 五、ZLMediaKit 侧配置

### 5.1 修改 ZLMediaKit `config.ini`

找到 `[hook]` 段，至少把以下三项改成指向 ArrowLive 服务：

```ini
[hook]
enable=1
timeoutSec=10
admin_params=secret=035c73f7-bb6b-4889-a715-d9eb2d1925cc

on_publish=http://<ArrowLive 地址>:8080/webhook/on_publish
on_stream_changed=http://<ArrowLive 地址>:8080/webhook/on_stream_changed
on_stream_none_reader=http://<ArrowLive 地址>:8080/webhook/on_stream_none_reader
```

推荐把其他 hook 也指过来，便于日志完整：

```ini
on_play=http://<ArrowLive 地址>:8080/webhook/on_play
on_stream_not_found=http://<ArrowLive 地址>:8080/webhook/on_stream_not_found
on_record_mp4=http://<ArrowLive 地址>:8080/webhook/on_record_mp4
on_server_started=http://<ArrowLive 地址>:8080/webhook/on_server_started
on_server_keepalive=http://<ArrowLive 地址>:8080/webhook/on_server_keepalive
on_flow_report=http://<ArrowLive 地址>:8080/webhook/on_flow_report
on_rtsp_auth=http://<ArrowLive 地址>:8080/webhook/on_rtsp_auth
on_rtsp_realm=http://<ArrowLive 地址>:8080/webhook/on_rtsp_realm
on_shell_login=http://<ArrowLive 地址>:8080/webhook/on_shell_login
on_http_access=http://<ArrowLive 地址>:8080/webhook/on_http_access
on_rtp_server_timeout=http://<ArrowLive 地址>:8080/webhook/on_rtp_server_timeout
```

### 5.2 确认 secret 一致

ZLMediaKit `config.ini` 的 `[api]` 段：

```ini
[api]
secret=035c73f7-bb6b-4889-a715-d9eb2d1925cc
```

这个值**必须**与 ArrowLive `config.yaml` 的 `zlmediakit.secret` 完全一致，否则 ArrowLive 调 ZLM API 会返回鉴权失败。

### 5.3 重启 ZLMediaKit

修改完 `config.ini` 后重启 ZLMediaKit 进程使配置生效。

### 5.4 验证 WebHook 连通

可选：手动 curl 一下 ArrowLive 的 webhook 端点，看是否能收到响应：

```bash
curl -X POST http://localhost:8080/webhook/on_server_keepalive \
  -H "Content-Type: application/json" \
  -d '{"mediaServerId":"test"}'
# 期望返回 {"code":0,"msg":"success"}
```

## 六、用户端使用

### 6.1 生成推流码

1. 浏览器访问 `http://<ArrowLive 地址>:8080/`
2. 在「房间名」输入框填入一个名字
   - 规则：1~32 字符，只允许字母、数字、下划线、短横线（正则 `^[a-zA-Z0-9_-]{1,32}$`）
   - 例：`room-01`、`test_abc`、`MyRoom`
3. 点「生成推流码」
4. 成功后页面展示三项关键信息：
   - **推流地址 (RTMP URL)**：完整带 token 的 RTMP URL，如 `rtmp://127.0.0.1:1935/live/room-01?token=a1b2...`
   - **推流 Token**：16 字节 hex 字符串（方便需要拆分的场景）
   - **拉流地址 (HTTP-FLV)**：浏览器/第三方播放器使用

每个字段右侧有「复制」按钮。

### 6.2 同页预览

页面会自动嵌入一个 flv.js 播放器，目标就是上述「拉流地址」。推流成功后 1~3 秒内显示画面。右上角状态徽标含义：

| 状态 | 含义 |
|---|---|
| 等待推流 | 房间已创建，但 ZLM 还未收到推流数据 |
| 直播中 | ZLM 已确认接收到推流 |
| 已断开 | 推流曾经开始但现已结束 |

状态通过轮询 `/user/api/rooms/{name}/status` 每 3 秒更新一次。

### 6.3 历史记录

页面底部「历史房间」列表由浏览器 **LocalStorage** 维护（key：`arrowlive.rooms`），只在当前浏览器可见。每条记录包含房间名、token、推流/拉流地址、创建时间。

- 点「再次预览」可重新挂接播放器
- 点「删除」仅从 LocalStorage 移除，不影响后端数据
- 最多保留 20 条（超出后自动丢弃最旧的）

### 6.4 占用冲突

若房间名当前处于 `created` 或 `active` 状态（即还没落到 `inactive`），再次申请会返回 HTTP 409：

```json
{"error": "room is currently occupied"}
```

此时页面「生成推流码」按钮旁的红色提示会显示错误信息。等原推流方停止推流（状态变为 `inactive`）后即可重新领用同名。

## 七、管理端使用

### 7.1 登录

1. 浏览器访问 `http://<ArrowLive 地址>:8080/admin/`
2. 若未登录，页面首次 API 请求会收到 401，自动跳转 `/admin/login.html`
3. 用 `config.yaml` 里配置的 `admin.username` / `admin.password` 登录
4. 成功后跳回 `/admin/`，session 有效期 12 小时

登录态通过 httpOnly cookie `admin_session` 保持，内存 session 管理，重启服务会失效。

### 7.2 列表模式

默认进入列表模式。布局：

- **左侧**：大尺寸视频播放器
- **右侧**：活动房间列表（5 秒轮询刷新）

交互：
- 打开页面时自动播放列表中第一个房间
- 点击右侧列表中的任一房间，左侧播放器立即切换到该房间
- 当前选中的房间在列表中高亮（蓝底）
- 若当前选中的房间离线了，自动回退到剩余列表的第一个；全部离线则清屏

### 7.3 九宫格模式

点击顶部「九宫格模式」tab 切换。布局：

- **顶部**：组 tab 栏，按「第 N 组（X）」显示，X 为该组实际房间数
- **下方**：2×2 播放器网格，固定 4 格

分组逻辑（后端自动计算）：
- 按房间创建时间升序排序
- 每 4 个切一组
- 最后一组不足 4 个时，前端用白色虚线框占位

交互：
- 默认展示第 1 组
- 点击 tab 切换组
- 切换回列表模式会销毁所有九宫格播放器，避免背景占用带宽

### 7.4 登出

右上角「登出」按钮清除服务端 session 与 cookie，跳转回登录页。

## 八、OBS 推流配置

### 8.1 拆分推流 URL

OBS Studio 的「设置 → 推流」界面要求把 URL 拆成两部分：

| ArrowLive 返回的完整 URL | OBS 拆分 |
|---|---|
| `rtmp://127.0.0.1:1935/live/room-01?token=a1b2c3d4...` | **服务器**：`rtmp://127.0.0.1:1935/live` |
| | **串流密钥**：`room-01?token=a1b2c3d4...` |

**关键点**：OBS 的「串流密钥」字段允许包含 query string，所以 `?token=xxx` 必须跟在房间名后面，一起填进这一栏。

### 8.2 步骤

1. OBS → 设置 → 推流
2. **服务** 选「自定义…」
3. **服务器** 填 `rtmp://<ArrowLive 返回的 host:port>/<app>`
4. **串流密钥** 填 `<房间名>?token=<token>`
5. 确定 → 开始推流

### 8.3 推流失败的常见原因

- 串流密钥里漏了 `?token=xxx` → ZLM 会被 ArrowLive 的 `on_publish` 拒绝
- token 复制错误或带了多余空格 → 同上
- ZLMediaKit 的 RTMP 端口被防火墙拦截
- ArrowLive 与 ZLMediaKit 的 `secret` 不一致 → 管理端列表会为空

## 九、HTTP API 参考

### 9.1 用户端 API `/user/api/*`

#### `POST /user/api/rooms`

创建或续期房间。

**请求体**：
```json
{"name": "room-01"}
```

**响应 200**：
```json
{
  "name": "room-01",
  "token": "a1b2c3d4e5f6...",
  "status": "created",
  "rtmp_url": "rtmp://127.0.0.1:1935/live/room-01?token=a1b2c3d4e5f6...",
  "play_url": "http://127.0.0.1/live/room-01.flv"
}
```

**错误码**：
- `400`：房间名格式非法
- `409`：房间名当前被占用（`created` 或 `active` 状态）
- `500`：服务端错误

#### `GET /user/api/rooms/{name}/status`

查询房间状态（供前端轮询）。

**响应 200**：
```json
{
  "name": "room-01",
  "status": "active",
  "active": true
}
```

`status` 取值：`created` / `active` / `inactive`。

**错误码**：
- `400`：房间名格式非法
- `404`：房间不存在

### 9.2 管理端 API `/admin/api/*`

除 `login` 外均需登录（cookie）。

#### `POST /admin/api/login`

```json
{"username": "admin", "password": "changeme"}
```
成功返回 `{"ok":true}` 并 set-cookie `admin_session=...`。失败返回 401。

#### `POST /admin/api/logout`

清除 session 与 cookie。返回 `{"ok":true}`。

#### `GET /admin/api/rooms`

活动房间列表。权威数据源 = ZLMediaKit `getMediaList` ∩ sqlite `active` 房间。

**响应 200**：
```json
{
  "rooms": [
    {"name": "room-01", "play_url": "http://127.0.0.1/live/room-01.flv"},
    {"name": "room-02", "play_url": "http://127.0.0.1/live/room-02.flv"}
  ]
}
```

#### `GET /admin/api/groups`

按 4 个一组切分的房间列表。

**响应 200**：
```json
{
  "groups": [
    [
      {"name": "room-01", "play_url": "..."},
      {"name": "room-02", "play_url": "..."},
      {"name": "room-03", "play_url": "..."},
      {"name": "room-04", "play_url": "..."}
    ],
    [
      {"name": "room-05", "play_url": "..."}
    ]
  ]
}
```

### 9.3 WebHook 端点 `/webhook/*`

由 ZLMediaKit 主动 POST 调用，业务方一般无需直接请求。

| 路径 | 用途 | 回复语义 |
|---|---|---|
| `/webhook/on_publish` | 推流鉴权，校验 token | 不通过返 `code:-1` 拒绝；通过返开启 rtsp/rtmp/ts/fmp4、关闭 hls/mp4 的控制字段 |
| `/webhook/on_stream_changed` | 流注册/注销 → 更新 sqlite 状态 | 对回复不敏感 |
| `/webhook/on_stream_none_reader` | 无人观看 | 返 `close:false` 保持流不被关闭 |
| `/webhook/on_play` 等 | 其余所有 hook | 一律放行 `code:0, msg:success` |

## 十、数据存储与状态机

### 10.1 SQLite 表结构

单表 `rooms`：

| 列名 | 类型 | 说明 |
|---|---|---|
| `name` | TEXT PRIMARY KEY | 房间名 |
| `token` | TEXT | 推流 token |
| `status` | TEXT | `created` / `active` / `inactive` |
| `created_at` | INTEGER | Unix 时间戳（秒） |
| `active_at` | INTEGER | 最近变 active 的 Unix 时间戳，从未活动过为 0 |

### 10.2 状态机

```
        POST /user/api/rooms
        (房间名未被占用)
 ──────────────────────────────▶ ┌──────────┐
                                 │ created  │
                                 └────┬─────┘
                                      │
                 on_stream_changed    │
                 regist=true          │
                                      ▼
                                 ┌──────────┐   on_stream_changed
                                 │  active  │ ◀─────regist=true───┐
                                 └────┬─────┘                      │
                                      │                            │
                on_stream_changed     │                            │
                regist=false          │                            │
                                      ▼                            │
                                 ┌──────────┐                      │
                                 │ inactive │ ─────────────────────┘
                                 └────┬─────┘
                                      │
                    POST /user/api/rooms
                    (同名再次申请，允许)
                                      │
                                      ▼
                                  (回到 created)
```

**占用判定规则**：
- `created` 与 `active` 都视为「被占用」，禁止他人创建同名房间
- 只有 `inactive` 状态的房间名可以被任何人再次领用（会覆盖原 token）

### 10.3 持久化位置

默认路径 `./data.db`（相对服务启动目录）。可通过 `database.path` 配置更改。

备份只需复制该 `.db` 文件；恢复就把文件放回原位即可。

### 10.4 管理员端的「活动房间」为什么要用交集

sqlite 中的 `active` 状态完全依赖 ZLMediaKit 的 `on_stream_changed` 回调。如果：
- ZLMediaKit 进程崩溃重启，未发 `regist=false` 回调
- ArrowLive 自身重启时正好漏掉一次回调
- 网络抖动导致回调丢失

这些情况会让 sqlite 里残留一些「幽灵 active 房间」。管理端列表若只看 sqlite 就会显示实际不存在的房间。

因此 `/admin/api/rooms` 和 `/admin/api/groups` 都以 **ZLMediaKit `getMediaList` 为权威**，再与 sqlite `active` 做交集。对应代码见 `internal/api/admin.go` 的 `activeRoomNames()`。

## 十一、故障排查

### 11.1 启动即报 `zlmediakit.secret is required`

说明 `config.yaml` 里缺少或留空了 `zlmediakit.secret`。该字段为必填。

### 11.2 启动报 `admin.username and admin.password are required`

`admin.username` 或 `admin.password` 留空了。两者都必填。

### 11.3 用户端「生成推流码」返回 409

该房间名当前处于 `created` 或 `active` 状态。两种解决方案：
- 换一个房间名
- 等原推流方停止推流（等几秒 ZLM 发 `on_stream_changed regist=false` 回调）

### 11.4 OBS 提示「无法连接到服务器」

检查：
- ArrowLive 和 ZLMediaKit 是否都在运行
- OBS 里的「服务器」字段格式是否为 `rtmp://host:port/<app>`，末尾不要多斜杠
- `rtmp_push_base` + `app` 拼出的地址是否 OBS 所在机器能访问到

### 11.5 OBS 能推，但 ArrowLive 页面预览黑屏

- 浏览器开发者工具 Network 面板查看 `*.flv` 请求，若返回 404 → ZLM 没收到推流，检查 OBS 状态和服务器日志
- 若返回 200 但无数据 → 检查 ZLMediaKit 的 HTTP-FLV 模块是否启用
- 浏览器 Console 有无 flv.js 报错
- `http_flv_play_base` 配置是否正确，浏览器能否直接访问该 URL

### 11.6 OBS 能推，但房间状态卡在「等待推流」

说明 ZLMediaKit 没调到 `on_publish` 或调了但被拒绝。检查：
- ZLMediaKit `config.ini` 的 `on_publish` 是否正确指向 ArrowLive
- ArrowLive 能否被 ZLMediaKit 访问到（防火墙 / 跨网段）
- ArrowLive 日志是否打印出推流被拒绝的原因（token mismatch / room not found 等）
- OBS 的串流密钥里 `?token=xxx` 是否完整无误

### 11.7 管理端登录后看不到任何房间

- 先通过浏览器直接访问 `http://<zlm>/index/api/getMediaList?secret=<secret>` 确认 ZLM 那里能看到活动流
- 若 ZLM 有流但 ArrowLive 管理端为空 → 检查 `zlmediakit.secret` 是否与 ZLM 一致、`zlmediakit.app` 是否匹配推流时用的 app
- 查看 ArrowLive 日志是否有 `admin: list rooms: ...` 错误输出

### 11.8 SQLite 文件被锁住

本项目配置 `SetMaxOpenConns(1)` 串行化访问，一般不会遇到。若仍出现，排查是否有其他进程占用同一 `data.db`。

## 十二、部署建议

### 12.1 进程管理

Linux 下建议用 systemd 管理：

```ini
# /etc/systemd/system/arrowlive.service
[Unit]
Description=ArrowLive
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/arrowlive
ExecStart=/opt/arrowlive/arrowlive -config /opt/arrowlive/config.yaml
Restart=on-failure
User=arrowlive

[Install]
WantedBy=multi-user.target
```

Windows 下可用 NSSM 或 WinSW 注册为服务。

### 12.2 反向代理与 HTTPS

项目本身不内置 HTTPS。建议在前面架 Nginx 或 Caddy：

```nginx
server {
    listen 443 ssl http2;
    server_name live.example.com;
    # ssl_certificate / ssl_certificate_key ...

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

此时 `http_flv_play_base` 应改为 `https://live.example.com`，但 RTMP 推流仍走明文 1935 端口（或由 ZLMediaKit 自行支持 RTMPS）。

### 12.3 数据备份

只需备份两样东西：
- `config.yaml`
- `data.db`

例：每日 cron 打包并上传：

```bash
tar czf arrowlive-$(date +%F).tar.gz config.yaml data.db1
```

### 12.4 升级流程

1. 停止服务
2. 备份 `data.db`
3. 替换二进制或拉新代码重新 `go build`
4. 启动服务

SQLite schema 目前只有单表且启动时 `CREATE TABLE IF NOT EXISTS`，升级无需手工迁移。未来若引入 schema 变更需要考虑迁移脚本。

### 12.5 资源占用

在典型小规模场景（同时 20~50 个活动房间，每路 1~2Mbps）：
- Go 进程内存占用通常 <100MB
- SQLite 文件体积 <1MB
- CPU 占用非常低（仅做 HTTP 路由和转发，媒体流由 ZLMediaKit 处理）

真正的带宽和 CPU 压力在 ZLMediaKit 端，按其部署规模规划即可。

---
