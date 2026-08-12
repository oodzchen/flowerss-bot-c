package handler

import (
	"context"
	"strings"

	"github.com/spf13/cast"
	tb "gopkg.in/telebot.v3"

	"github.com/indes/flowerss-bot/internal/bot/message"
	"github.com/indes/flowerss-bot/internal/bot/session"
	"github.com/indes/flowerss-bot/internal/core"
	"github.com/indes/flowerss-bot/internal/log"
)

type SetFeedTitle struct {
	core *core.Core
}

func NewSetFeedTitle(core *core.Core) *SetFeedTitle {
	return &SetFeedTitle{core: core}
}

func (s *SetFeedTitle) Command() string {
	return "/setfeedtitle"
}

func (s *SetFeedTitle) Description() string {
	return "设置RSS订阅标题"
}

func (s *SetFeedTitle) getMessageWithoutMention(ctx tb.Context) string {
	mention := message.MentionFromMessage(ctx.Message())
	if mention == "" {
		return ctx.Message().Payload
	}
	return strings.Replace(ctx.Message().Payload, mention, "", 1)
}

func (s *SetFeedTitle) Handle(ctx tb.Context) error {
	args := strings.Fields(s.getMessageWithoutMention(ctx))
	if len(args) == 0 {
		return ctx.Reply("/setfeedtitle [sourceID] [新标题] 设置订阅标题；省略新标题可恢复RSS原标题")
	}

	sourceID := cast.ToUint(args[0])
	if sourceID == 0 {
		return ctx.Reply("请输入正确的RSS源ID")
	}

	subscribeUserID := ctx.Chat().ID
	mentionChat, _ := session.GetMentionChatFromCtxStore(ctx)
	if mentionChat != nil {
		subscribeUserID = mentionChat.ID
	}

	title := strings.Join(args[1:], " ")
	if err := s.core.SetSubscriptionTitle(context.Background(), subscribeUserID, sourceID, title); err != nil {
		log.Errorf("SetSubscriptionTitle failed, %v", err)
		return ctx.Reply("订阅标题设置失败!")
	}
	if title == "" {
		return ctx.Reply("已恢复RSS原标题!")
	}
	return ctx.Reply("订阅标题设置成功!")
}

func (s *SetFeedTitle) Middlewares() []tb.MiddlewareFunc {
	return nil
}
