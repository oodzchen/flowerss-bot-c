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
	"github.com/indes/flowerss-bot/internal/timezone"
)

type SetTZ struct {
	core *core.Core
}

func NewSetTZ(core *core.Core) *SetTZ {
	return &SetTZ{core: core}
}

func (s *SetTZ) Command() string {
	return "/settz"
}

func (s *SetTZ) Description() string {
	return "设置订阅推送时间显示时区"
}

const setTZUsage = `用法：
/settz <时区> 设置当前会话所有订阅的时区
/settz <sourceID> <时区> 只设置某一个订阅
/settz off 重置为默认时区
常见时区示例：Asia/Shanghai(北京时间) UTC America/New_York Europe/London +08:00 -05:00`

func (s *SetTZ) Handle(ctx tb.Context) error {
	payload := ctx.Message().Payload
	mention := message.MentionFromMessage(ctx.Message())
	if mention != "" {
		payload = strings.Replace(payload, mention, "", 1)
	}
	args := strings.Fields(payload)
	if len(args) == 0 {
		return ctx.Reply(setTZUsage)
	}

	subscribeUserID := ctx.Chat().ID
	mentionChat, _ := session.GetMentionChatFromCtxStore(ctx)
	if mentionChat != nil {
		subscribeUserID = mentionChat.ID
	}

	sourceID := uint(0)
	rawTZ := ""
	if id := cast.ToUint(args[0]); id != 0 {
		sourceID = id
		if len(args) < 2 {
			return ctx.Reply("请指定时区，例如：/settz 12 Asia/Shanghai 或 /settz 12 +08:00")
		}
		rawTZ = args[1]
	} else {
		rawTZ = args[0]
	}

	tzStr, err := timezone.NormalizeTimezone(rawTZ)
	if err != nil {
		return ctx.Reply("无效的时区格式！示例：Asia/Shanghai、UTC、+08:00、-05:00，或输入 off 重置。")
	}

	if sourceID != 0 {
		if err := s.core.SetSubscriptionTimezone(context.Background(), subscribeUserID, sourceID, tzStr); err != nil {
			log.Errorf("SetSubscriptionTimezone failed, %v", err)
			return ctx.Reply("时区设置失败!")
		}
		if tzStr == "" {
			return ctx.Reply("已重置该订阅的时区为默认!")
		}
		return ctx.Reply(fmt.Sprintf("已为该订阅设置推送时区: %s", tzStr))
	}

	count, err := s.core.SetSubscriptionTimezoneForAll(context.Background(), subscribeUserID, tzStr)
	if err != nil {
		log.Errorf("SetSubscriptionTimezoneForAll failed, %v", err)
		return ctx.Reply("时区设置失败!")
	}
	if tzStr == "" {
		return ctx.Reply(fmt.Sprintf("已重置 %d 个订阅的时区为默认", count))
	}
	return ctx.Reply(fmt.Sprintf("已为 %d 个订阅设置推送时区: %s", count, tzStr))
}

func (s *SetTZ) Middlewares() []tb.MiddlewareFunc {
	return nil
}
