package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cast"
	tb "gopkg.in/telebot.v3"

	"github.com/indes/flowerss-bot/internal/bot/message"
	"github.com/indes/flowerss-bot/internal/bot/session"
	"github.com/indes/flowerss-bot/internal/core"
	"github.com/indes/flowerss-bot/internal/log"
)

type SetPreview struct {
	core *core.Core
}

func NewSetPreview(core *core.Core) *SetPreview {
	return &SetPreview{core: core}
}

func (s *SetPreview) Command() string {
	return "/setpreview"
}

func (s *SetPreview) Description() string {
	return "设置订阅正文预览字符数与截取方向"
}

const setPreviewUsage = `用法：
/setpreview <字符数> 设置当前会话所有订阅的预览字数与截取方向
/setpreview <sourceID> <字符数> 只设置某一个订阅
/setpreview 0 或 /setpreview off 关闭预览
/setpreview default 或 /setpreview reset 重置为全局默认配置

参数说明：
正数 (如 400)：截取正文前 400 个字符
负数 (如 -400)：截取正文后 400 个字符
0 或 off：关闭正文预览
default 或 reset：恢复全局默认配置`

func parsePreviewArg(raw string) (*int, error) {
	norm := strings.ToLower(strings.TrimSpace(raw))
	if norm == "default" || norm == "reset" || norm == "默认" || norm == "重置" {
		return nil, nil
	}
	if norm == "off" || norm == "关闭" || norm == "none" {
		val := 0
		return &val, nil
	}

	num, err := strconv.Atoi(norm)
	if err != nil {
		return nil, fmt.Errorf("invalid preview number: %s", raw)
	}
	return &num, nil
}

func (s *SetPreview) Handle(ctx tb.Context) error {
	payload := ctx.Message().Payload
	mention := message.MentionFromMessage(ctx.Message())
	if mention != "" {
		payload = strings.Replace(payload, mention, "", 1)
	}
	args := strings.Fields(payload)
	if len(args) == 0 {
		return ctx.Reply(setPreviewUsage)
	}

	subscribeUserID := ctx.Chat().ID
	mentionChat, _ := session.GetMentionChatFromCtxStore(ctx)
	if mentionChat != nil {
		subscribeUserID = mentionChat.ID
	}

	sourceID := uint(0)
	var rawLimit string

	if len(args) >= 2 {
		id := cast.ToUint(args[0])
		if id == 0 {
			return ctx.Reply("无效的 RSS 源 ID！例如：/setpreview 12 400 或 /setpreview 12 -400")
		}
		sourceID = id
		rawLimit = args[1]
	} else {
		rawLimit = args[0]
	}

	length, err := parsePreviewArg(rawLimit)
	if err != nil {
		return ctx.Reply("无效的预览字符数格式！\n正数（如 400）截取前 400 字符，负数（如 -400）截取后 400 字符，0 或 off 关闭，default 重置为默认。")
	}

	if sourceID != 0 {
		if err := s.core.SetSubscriptionPreviewLength(context.Background(), subscribeUserID, sourceID, length); err != nil {
			log.Errorf("SetSubscriptionPreviewLength failed, %v", err)
			return ctx.Reply("预览字符数设置失败!")
		}
		if length == nil {
			return ctx.Reply("已重置该订阅的预览字符数为全局默认配置!")
		}
		if *length == 0 {
			return ctx.Reply("已关闭该订阅的正文预览!")
		}
		if *length > 0 {
			return ctx.Reply(fmt.Sprintf("已为该订阅设置正文预览: 前 %d 字符", *length))
		}
		return ctx.Reply(fmt.Sprintf("已为该订阅设置正文预览: 后 %d 字符", -*length))
	}

	count, err := s.core.SetSubscriptionPreviewLengthForAll(context.Background(), subscribeUserID, length)
	if err != nil {
		log.Errorf("SetSubscriptionPreviewLengthForAll failed, %v", err)
		return ctx.Reply("预览字符数设置失败!")
	}
	if length == nil {
		return ctx.Reply(fmt.Sprintf("已重置 %d 个订阅的预览字符数为全局默认配置", count))
	}
	if *length == 0 {
		return ctx.Reply(fmt.Sprintf("已关闭 %d 个订阅的正文预览", count))
	}
	if *length > 0 {
		return ctx.Reply(fmt.Sprintf("已为 %d 个订阅设置正文预览: 前 %d 字符", count, *length))
	}
	return ctx.Reply(fmt.Sprintf("已为 %d 个订阅设置正文预览: 后 %d 字符", count, -*length))
}

func (s *SetPreview) Middlewares() []tb.MiddlewareFunc {
	return nil
}
