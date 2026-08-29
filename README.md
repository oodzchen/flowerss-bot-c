# flowerss bot

[![Build Status](https://github.com/indes/flowerss-bot/workflows/Release/badge.svg)](https://github.com/indes/flowerss-bot/actions?query=workflow%3ARelease)
[![Test Status](https://github.com/indes/flowerss-bot/workflows/Test/badge.svg)](https://github.com/indes/flowerss-bot/actions?query=workflow%3ATest)
![Build Docker Image](https://github.com/indes/flowerss-bot/workflows/Build%20Docker%20Image/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/indes/flowerss-bot)](https://goreportcard.com/report/github.com/indes/flowerss-bot)
![GitHub](https://img.shields.io/github/license/indes/flowerss-bot.svg)

[安装与使用文档](https://flowerss-bot.now.sh/)

<img src="https://github.com/rssflow/img/raw/master/images/rssflow_demo.gif" width = "300"/>

## Features

- 订阅 RSS/Atom 源并自动推送更新
- 支持私聊、Group 和 Channel 订阅，群组和频道操作会检查管理员权限
- 支持 Telegram 应用内 Instant View（Telegraph）和纯文字正文预览
- 支持自定义订阅标题、标签、抓取频率、通知和 Telegraph 开关
- 支持按订阅设置推送翻译语言和发布时间显示时区
- 支持 OPML 批量导入、导出
- 支持自定义 HTML/Markdown 推送模板
- 支持 SQLite 和 MySQL，可配置 SOCKS5 代理、自定义 Telegram Bot API 和用户白名单

## 安装

运行前需要通过 [@BotFather](https://t.me/BotFather) 创建 Telegram Bot 并取得 Bot Token。

### Docker 部署

创建配置和数据目录：

```bash
mkdir -p ~/flowerss
wget -O ~/flowerss/config.yml \
  https://raw.githubusercontent.com/indes/flowerss-bot/master/config.yml.sample
```

编辑 `~/flowerss/config.yml`，至少填写 `bot_token`，并将 SQLite 路径改为容器内路径：

```yaml
bot_token: 123456:telegram-bot-token

sqlite:
  path: /root/.flowerss/data.db
```

启动容器：

```bash
docker run -d \
  --name flowerss-bot \
  --restart unless-stopped \
  -v ~/flowerss:/root/.flowerss \
  indes/flowerss-bot
```

项目中的 `docker-compose.yml` 也可以用于本地构建，配置文件需要放在 `./conf/config.yml`：

```bash
docker compose up -d --build
```

### 二进制部署

从 [Releases](https://github.com/indes/flowerss-bot/releases) 下载对应平台的版本，将 `config.yml` 放在程序工作目录后运行：

```bash
./flowerss-bot
```

也可以指定工作目录或配置文件：

```bash
./flowerss-bot -d /path/to/workdir
./flowerss-bot -c /path/to/config.yml
```

### 源码编译部署

需要 Go 1.18 或更高版本：

```bash
git clone https://github.com/indes/flowerss-bot.git
cd flowerss-bot
make build
cp config.yml.sample config.yml
./flowerss-bot
```

## 配置

复制 [config.yml.sample](config.yml.sample) 为 `config.yml`，配置示例：

```yaml
bot_token: 123456:telegram-bot-token
update_interval: 10
error_threshold: 100
preview_text: 300
disable_web_page_preview: false

sqlite:
  path: ./data.db

log:
  level: release
  db_log: false
```

配置说明：

| 配置项 | 含义 | 默认值 |
| --- | --- | --- |
| `bot_token` | Telegram Bot Token，必填 | 无 |
| `telegraph_token` | Telegraph Token，可填写一个或多个 | 不启用 |
| `telegraph_account` | 未提供 Token 时用于创建 Telegraph 账号的名称 | 不启用 |
| `telegraph_author_name` | Telegraph 页面作者名称 | `flowerss-bot` |
| `telegraph_author_url` | Telegraph 页面作者链接 | 空 |
| `preview_text` | 推送正文预览字符数，正数为截取前 N 字符，负数为截取后 N 字符，设为 `0` 可关闭 | `300` |
| `disable_web_page_preview` | 是否关闭 Telegram 网页预览 | `false` |
| `update_interval` | 全局扫描间隔及新订阅的默认抓取频率，单位为分钟 | `10` |
| `error_threshold` | RSS 源连续抓取失败多少次后暂停更新 | `100` |
| `user_agent` | 抓取 RSS 时使用的 User-Agent | 空 |
| `socks5` | SOCKS5 代理地址，如 `127.0.0.1:1080` | 空 |
| `telegram.endpoint` | 自定义 Telegram Bot API 地址 | Telegram 官方 API |
| `allowed_users` | 允许操作 Bot 的 Telegram 用户 ID 列表 | 不限制 |
| `sqlite.path` | SQLite 数据库文件路径 | 工作目录下的 `data.db` |
| `mysql.*` | MySQL 的主机、端口、用户、密码和数据库；配置 `mysql.host` 后优先使用 MySQL | 不启用 |
| `message_mode` | 推送模板模式，可设为 `html`、`markdown` 或 `md` | `html` |
| `message_tpl` | Go Template 格式的推送模板 | 内置模板 |
| `log.level` | 日志级别，设为 `debug` 可输出调试日志 | `release` |
| `log.file` | 同时写入的日志文件路径 | 仅输出到标准错误 |
| `log.db_log` | 是否输出数据库调试日志 | `false` |

如果配置多个 Telegraph Token，可以使用数组格式分散接口请求：

```yaml
telegraph_token:
  - token_1
  - token_2
```

也可以只配置账号信息，由程序创建 Telegraph 账号。首次启动后请从日志保存生成的 Token，以便后续继续使用同一账号：

```yaml
telegraph_account: flowerss
telegraph_author_name: flowerss-bot
telegraph_author_url: https://github.com/indes/flowerss-bot
```

## 安装与使用

使用命令：

```text
/sub <url> 订阅 RSS 源
/unsub [url] 取消订阅；省略 URL 时从列表中选择
/list 查看当前订阅
/set 设置单个订阅的抓取、通知、Telegraph 和标签开关
/setfeedtag <sourceID> <tag1> [tag2] [tag3] 设置订阅标签，最多三个
/setfeedtitle <sourceID> [title] 设置订阅标题，省略 title 恢复 RSS 原标题
/setinterval <interval> <sourceID> [sourceID...] 设置订阅刷新频率，单位为分钟
/setlang <lang> 设置全部订阅的翻译语言
/setlang <sourceID> <lang> 设置单个订阅的翻译语言，lang 为 off 时关闭
/settz <timezone> 设置全部订阅的推送时间显示时区
/settz <sourceID> <timezone> 设置单个订阅的时区，timezone 为 off 时重置
/setpreview <count> 设置全部订阅的正文预览字符数与截取方向（正数前N字，负数后N字，0/off关闭，default重置）
/setpreview <sourceID> <count> 设置单个订阅的正文预览字符数与截取方向
/translate <lang> <text> 测试翻译服务
/activeall 开启所有订阅源的抓取
/pauseall 暂停所有订阅源的抓取
/import 查看 OPML 导入说明，按提示发送 OPML 文件
/export 导出 OPML 文件
/unsuball 取消所有订阅
/version 查看 Bot 版本
/help 查看帮助
```

`sourceID` 可以通过 `/list` 查看。执行 `/set` 后选择订阅，可以通过按钮暂停或重启抓取、开关静默通知、开关 Telegraph 转码以及设置标签。

### Group 订阅

将 Bot 加入 Group 后即可在群内使用普通命令。涉及订阅变更和设置的操作仅允许群管理员执行，推送消息会发送到当前 Group。

### Channel 订阅

1. 将 Bot 添加为 Channel 管理员；
2. 操作者也需要是该 Channel 的管理员；
3. 在与 Bot 的对话中，通过 `@ChannelID` 指定要管理的频道。

常用命令：

```text
/sub @ChannelID <url>
/unsub @ChannelID <url>
/list @ChannelID
/setfeedtag @ChannelID <sourceID> <tag1> [tag2] [tag3]
/setfeedtitle @ChannelID <sourceID> [title]
/setinterval @ChannelID <interval> <sourceID> [sourceID...]
/setlang @ChannelID <lang>
/setlang @ChannelID <sourceID> <lang>
/settz @ChannelID <timezone>
/settz @ChannelID <sourceID> <timezone>
/setpreview @ChannelID <count>
/setpreview @ChannelID <sourceID> <count>
/activeall @ChannelID
/pauseall @ChannelID
/export @ChannelID
```

为 Channel 导入 OPML 时，直接发送 `.opml` 文件并在文件说明中附上 `@ChannelID`。

`ChannelID` 是频道的公开用户名。Private Channel 可以临时设为 Public，完成设置后再改回 Private，不影响后续推送。

例如为 `t.me/debug` 频道订阅 RSS：

```text
/sub @debug https://www.ruanyifeng.com/blog/atom.xml
```

## 推送翻译（可选）

推送前可把 RSS 标题和预览正文翻译为目标语言。翻译使用 OpenAI 兼容的 Chat Completions 接口，支持 DeepSeek、OpenAI、OpenRouter、Ollama 等后端。接口不可用或翻译失败时会回退到原文，不影响正常推送。

DeepSeek 示例：

```yaml
translate:
  provider: llm
  base_url: https://api.deepseek.com/v1
  api_key: sk-xxx
  model: deepseek-chat
```

OpenRouter 示例：

```yaml
translate:
  provider: openrouter
  base_url: https://openrouter.ai/api/v1
  api_key: sk-or-v1-xxx
  model: anthropic/claude-3.5-sonnet
  # http_referer: https://your-site.example
  # x_title: flowerss-bot
```

本地 Ollama 示例：

```yaml
translate:
  provider: llm
  base_url: http://127.0.0.1:11434/v1
  api_key: ""
  model: qwen3:8b
```

使用 `/setlang zh` 为当前会话的全部订阅开启中文翻译，或使用 `/setlang 12 zh` 只设置 ID 为 `12` 的订阅。常用语言代码包括 `zh`、`en`、`ja`、`ko`、`fr`、`de`、`ru`、`es`，也可以直接使用其他语言代码。

翻译不生效时排查：

1. 启动日志应出现 `init translate: provider=... model=... base_url=... api_key=true/false`，确认配置被正确读取；
2. 使用 `/translate zh Hello world` 测试服务连通性，错误信息会包含 HTTP 状态码和接口返回内容；
3. 推送日志中的 `translation enabled subscribers` 应大于 `0`，否则需要先使用 `/setlang`；
4. 将 `log.level` 临时改为 `debug`，可以查看翻译请求、缓存命中和失败回退日志。

## 推送时间时区

RSS 提供发布时间时，默认按解析出的时区显示。可以为全部订阅或单个订阅指定 IANA 时区、UTC/GMT 偏移：

```text
/settz Asia/Shanghai
/settz 12 America/New_York
/settz +08:00
/settz 12 off
```

支持 `Asia/Shanghai`、`America/New_York`、`Europe/London`、`UTC`、`+08:00`、`UTC+8`、`-05:00` 等格式。使用 `off`、`reset` 或 `clear` 可以恢复默认时区。

## 自定义推送模板

`message_tpl` 使用 Go Template 语法，可使用以下字段：

| 字段 | 含义 |
| --- | --- |
| `SourceTitle` | 订阅标题，优先使用 `/setfeedtitle` 设置的标题 |
| `ContentTitle` | RSS 条目标题，启用翻译后为翻译结果 |
| `RawLink` | 原文链接 |
| `PublishedAt` | 发布时间，格式为 `YYYY-MM-DD HH:mm:ss ±时区` |
| `PreviewText` | 正文预览，启用翻译后为翻译结果 |
| `TelegraphURL` | Telegraph 页面链接 |
| `Tags` | 订阅标签 |
| `EnableTelegraph` | 当前消息是否有可用的 Telegraph 页面 |

修改模板后可以在启动 Bot 前检查语法和渲染结果：

```bash
./flowerss-bot -c config.yml -testtpl
```

## 常见问题

### 如何申请 Telegraph Token？

如果需要 Telegram 应用内 Instant View，可以执行：

```bash
curl "https://api.telegra.ph/createAccount?short_name=flowerss&author_name=flowerss&author_url=https://github.com/indes/flowerss-bot"
```

返回 JSON 中的 `access_token` 即为 Telegraph Token。

### Telegraph 出现 `FLOOD_WAIT` 错误怎么办？

这是创建 Telegraph 页面过快触发了接口限制。可以在配置文件中添加多个 `telegraph_token` 分散请求，或降低 RSS 抓取频率。

### RSS 源为什么停止更新？

RSS 源连续抓取失败达到 `error_threshold` 后会自动暂停，并向订阅者发送提示。源恢复后，可以通过 `/set` 中的“重启更新”按钮或 `/activeall` 重新开启抓取。

详细使用方法请查阅项目[使用文档](https://flowerss-bot.now.sh/#/usage)。
