package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cast"
	tb "gopkg.in/telebot.v3"

	"github.com/indes/flowerss-bot/internal/bot/message"
	"github.com/indes/flowerss-bot/internal/bot/session"
	"github.com/indes/flowerss-bot/internal/core"
	"github.com/indes/flowerss-bot/internal/log"
	"github.com/indes/flowerss-bot/internal/translate"
)

type SetLang struct {
	core *core.Core
}

func NewSetLang(core *core.Core) *SetLang {
	return &SetLang{core: core}
}

func (s *SetLang) Command() string {
	return "/setlang"
}

func (s *SetLang) Description() string {
	return "设置订阅推送翻译语言"
}

const setLangUsage = `用法：
/setlang <语言代码> 设置当前会话所有订阅的翻译语言
/setlang <sourceID> <语言代码> 只设置某一个订阅
/setlang off 关闭翻译
常用语言代码：zh(中文) en(英文) ja(日文) ko(韩文) fr(法文) de(德文) ru(俄文) es(西文)`

func (s *SetLang) Handle(ctx tb.Context) error {
	payload := ctx.Message().Payload
	mention := message.MentionFromMessage(ctx.Message())
	if mention != "" {
		payload = strings.Replace(payload, mention, "", 1)
	}
	args := strings.Fields(payload)
	if len(args) == 0 {
		return ctx.Reply(setLangUsage)
	}

	subscribeUserID := ctx.Chat().ID
	mentionChat, _ := session.GetMentionChatFromCtxStore(ctx)
	if mentionChat != nil {
		subscribeUserID = mentionChat.ID
	}

	// 第一个参数是数字时视为 sourceID：/setlang <sourceID> <lang>
	sourceID := uint(0)
	lang := ""
	if id := cast.ToUint(args[0]); id != 0 {
		sourceID = id
		if len(args) < 2 {
			return ctx.Reply("请指定语言代码，例如：/setlang 12 zh")
		}
		lang = translate.NormalizeLang(args[1])
	} else {
		lang = translate.NormalizeLang(args[0])
	}

	if sourceID != 0 {
		if err := s.core.SetSubscriptionLang(context.Background(), subscribeUserID, sourceID, lang); err != nil {
			log.Errorf("SetSubscriptionLang failed, %v", err)
			return ctx.Reply("翻译语言设置失败!")
		}
		if lang == "" {
			return ctx.Reply("已关闭该订阅的翻译!")
		}
		return ctx.Reply(fmt.Sprintf("已为该订阅设置翻译语言: %s (%s)", lang, translate.LanguageName(lang)))
	}

	count, err := s.core.SetSubscriptionLangForAll(context.Background(), subscribeUserID, lang)
	if err != nil {
		log.Errorf("SetSubscriptionLangForAll failed, %v", err)
		return ctx.Reply("翻译语言设置失败!")
	}
	if lang == "" {
		return ctx.Reply(fmt.Sprintf("已关闭 %d 个订阅的翻译", count))
	}
	return ctx.Reply(fmt.Sprintf("已为 %d 个订阅设置翻译语言: %s (%s)", count, lang, translate.LanguageName(lang)))
}

func (s *SetLang) Middlewares() []tb.MiddlewareFunc {
	return nil
}
