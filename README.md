# flowerss bot

[![Build Status](https://github.com/indes/flowerss-bot/workflows/Release/badge.svg)](https://github.com/indes/flowerss-bot/actions?query=workflow%3ARelease)
[![Test Status](https://github.com/indes/flowerss-bot/workflows/Test/badge.svg)](https://github.com/indes/flowerss-bot/actions?query=workflow%3ATest)
![Build Docker Image](https://github.com/indes/flowerss-bot/workflows/Build%20Docker%20Image/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/indes/flowerss-bot)](https://goreportcard.com/report/github.com/indes/flowerss-bot)
![GitHub](https://img.shields.io/github/license/indes/flowerss-bot.svg)

[安装与使用文档](https://flowerss-bot.now.sh/)  

<img src="https://github.com/rssflow/img/raw/master/images/rssflow_demo.gif" width = "300"/>

## Features

- 常见的 RSS Bot 该有的功能
- 支持 Telegram 应用内 instant view
- 支持为 Group 和 Channel 订阅 RSS 消息
- 丰富的订阅设置

## 安装与使用

详细安装与使用方法请查阅项目[使用文档](https://flowerss-bot.now.sh/)。  

使用命令：

```
/sub [url] 订阅（url 为可选）
/unsub [url] 取消订阅（url 为可选）
/list 查看当前订阅
/set 设置订阅
/check 检查当前订阅
/setfeedtag [sub id] [tag1] [tag2] 设置订阅标签（最多设置三个Tag，以空格分割）
/setfeedtitle [sub id] [title] 设置订阅标题（省略title恢复RSS原标题）
/setinterval [interval] [sub id] 设置订阅刷新频率（可设置多个sub id，以空格分割）
/setlang [lang] 设置订阅推送翻译语言（如 /setlang zh；/setlang [sub id] [lang] 只设置单个订阅；/setlang off 关闭）
/translate [lang] [text] 测试翻译服务是否可用（调试用，如 /translate zh Hello world）
/activeall 开启所有订阅
/pauseall 暂停所有订阅
/import 导入 OPML 文件
/export 导出 OPML 文件
/unsuball 取消所有订阅
/help 帮助
```
详细使用方法请查阅项目[使用文档](https://flowerss-bot.now.sh/#/usage)。

## 推送翻译（可选）

推送前可把 RSS 内容翻译为目标语言（标题 + 预览正文），基于 LLM 的 OpenAI 兼容接口，支持 DeepSeek / OpenAI / OpenRouter / Ollama 等任意后端，在 `config.yml` 的 `translate` 段配置：

```yaml
translate:
  provider: llm                     # 或 openrouter
  base_url: https://api.deepseek.com/v1
  api_key: sk-xxx
  model: deepseek-chat              # OpenRouter 填 厂商/模型，如 anthropic/claude-3.5-sonnet
```

OpenRouter 示例：

```yaml
translate:
  provider: openrouter
  base_url: https://openrouter.ai/api/v1
  api_key: sk-or-v1-xxx
  model: anthropic/claude-3.5-sonnet   # 在 openrouter.ai/models 可查到任意模型 ID
  # http_referer: https://your-site.example
  # x_title: flowerss-bot
```

用户用 `/setlang <语言代码>` 开启翻译（如 `/setlang zh`），语言代码支持 `zh en ja ko fr de ru es` 等。

翻译不生效时排查：

1. 启动日志应出现 `init translate: provider=... model=... base_url=... api_key=true/false`，确认配置被正确读取；
2. 推送时日志里 `translation enabled subscribers` 应大于 0，否则说明 `/setlang` 未生效；
3. 用 `/translate zh Hello world` 直接测试翻译服务连通性，报错信息会包含 HTTP 状态码和接口返回内容（如 401 密钥错误、404 模型名错误）；
4. 将 `log.level` 临时改为 `debug` 可看到每次翻译的完整过程日志（发起、命中缓存、成功/失败）。
