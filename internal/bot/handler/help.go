package handler

import (
	tb "gopkg.in/telebot.v3"
)

type Help struct {
}

func NewHelp() *Help {
	return &Help{}
}

func (h *Help) Command() string {
	return "/help"
}

func (h *Help) Description() string {
	return "帮助"
}

func (h *Help) Handle(ctx tb.Context) error {
	message := `
	命令：
	/sub 订阅源
	/unsub  取消订阅
	/list 查看当前订阅源
	/set 设置订阅
	/check 检查当前订阅
	/setfeedtag 设置订阅标签
	/setfeedtitle 设置订阅标题
	/setinterval 设置订阅刷新频率
	/setlang 设置订阅推送翻译语言（如 /setlang zh）
	/settz 设置订阅推送时间显示时区（如 /settz Asia/Shanghai 或 /settz +08:00）
	/setpreview 设置订阅正文预览字符数与截取方向（如 /setpreview 400 或 /setpreview -400）
	/translate 测试翻译服务（调试用，如 /translate zh Hello world）
	/activeall 开启所有订阅
	/pauseall 暂停所有订阅
	/help 帮助
	/import 导入 OPML 文件
	/export 导出 OPML 文件
	/unsuball 取消所有订阅
	详细使用方法请看：https://github.com/indes/flowerss-bot
	`
	return ctx.Send(message)
}

func (h *Help) Middlewares() []tb.MiddlewareFunc {
	return nil
}
