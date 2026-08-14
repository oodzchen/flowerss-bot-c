package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	tb "gopkg.in/telebot.v3"

	"github.com/indes/flowerss-bot/internal/bot/message"
	"github.com/indes/flowerss-bot/internal/log"
	"github.com/indes/flowerss-bot/internal/translate"
)

// Translate 是一个调试命令：直接调用翻译服务翻译一段文本，用于验证
// config.yml 里 translate 配置是否正确（API Key、模型名、网络连通性等）。
type Translate struct {
	translator translate.Translator
}

func NewTranslate(translator translate.Translator) *Translate {
	return &Translate{translator: translator}
}

func (h *Translate) Command() string {
	return "/translate"
}

func (h *Translate) Description() string {
	return "测试翻译（调试用）"
}

func (h *Translate) Handle(ctx tb.Context) error {
	if h.translator == nil {
		return ctx.Reply("翻译服务未启用：请检查 config.yml 的 translate 配置（provider/base_url/api_key/model）")
	}

	payload := ctx.Message().Payload
	mention := message.MentionFromMessage(ctx.Message())
	if mention != "" {
		payload = strings.Replace(payload, mention, "", 1)
	}
	args := strings.Fields(payload)
	if len(args) < 2 {
		return ctx.Reply("用法：/translate <语言代码> <文本>\n例如：/translate zh Hello world")
	}

	lang := translate.NormalizeLang(args[0])
	if lang == "" {
		return ctx.Reply("请输入有效的语言代码，例如 zh / en / ja / ko")
	}
	text := strings.Join(args[1:], " ")

	ctx2, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	translated, err := h.translator.Translate(ctx2, text, lang)
	if err != nil {
		log.Errorf("translate test failed, lang %s: %v", lang, err)
		return ctx.Reply(fmt.Sprintf("翻译失败（%s → %s）：\n%v", text, translate.LanguageName(lang), err))
	}
	log.Infof("translate test ok, lang %s: %q -> %q", lang, text, translated)
	return ctx.Reply(fmt.Sprintf("翻译结果（%s → %s）：\n%s", text, translate.LanguageName(lang), translated))
}

func (h *Translate) Middlewares() []tb.MiddlewareFunc {
	return nil
}
