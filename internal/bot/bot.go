package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	tb "gopkg.in/telebot.v3"

	"github.com/indes/flowerss-bot/internal/bot/handler"
	"github.com/indes/flowerss-bot/internal/bot/middleware"
	"github.com/indes/flowerss-bot/internal/bot/preview"
	"github.com/indes/flowerss-bot/internal/config"
	"github.com/indes/flowerss-bot/internal/core"
	"github.com/indes/flowerss-bot/internal/log"
	"github.com/indes/flowerss-bot/internal/model"
	"github.com/indes/flowerss-bot/internal/timezone"
	"github.com/indes/flowerss-bot/internal/translate"
)

type Bot struct {
	core       *core.Core
	tb         *tb.Bot // telebot.Bot instance
	translator translate.Translator
	transCache *translate.Cache
}

func NewBot(core *core.Core) *Bot {
	log.Infof("init telegram bot, token %s, endpoint %s", config.BotToken, config.TelegramEndpoint)
	settings := tb.Settings{
		URL:    config.TelegramEndpoint,
		Token:  config.BotToken,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
		Client: core.HttpClient().Client(),
	}

	logLevel := config.GetString("log.level")
	if strings.ToLower(logLevel) == "debug" {
		settings.Verbose = true
	}

	b := &Bot{
		core:       core,
		translator: translate.NewFromConfig(core.HttpClient()),
		transCache: translate.NewCache(2000),
	}

	var err error
	b.tb, err = tb.NewBot(settings)
	if err != nil {
		log.Error(err)
		return nil
	}
	b.tb.Use(middleware.UserFilter(), middleware.PreLoadMentionChat(), middleware.IsChatAdmin())
	return b
}

func (b *Bot) registerCommands(appCore *core.Core) error {
	commandHandlers := []handler.CommandHandler{
		handler.NewStart(),
		handler.NewPing(b.tb),
		handler.NewAddSubscription(appCore),
		handler.NewRemoveSubscription(b.tb, appCore),
		handler.NewListSubscription(appCore),
		handler.NewRemoveAllSubscription(),
		handler.NewOnDocument(b.tb, appCore),
		handler.NewSet(b.tb, appCore),
		handler.NewSetFeedTag(appCore),
		handler.NewSetFeedTitle(appCore),
		handler.NewSetUpdateInterval(appCore),
		handler.NewSetLang(appCore),
		handler.NewSetTZ(appCore),
		handler.NewSetPreview(appCore),
		handler.NewTranslate(b.translator),
		handler.NewExport(appCore),
		handler.NewImport(),
		handler.NewPauseAll(appCore),
		handler.NewActiveAll(appCore),
		handler.NewHelp(),
		handler.NewVersion(),
	}

	for _, h := range commandHandlers {
		b.tb.Handle(h.Command(), h.Handle, h.Middlewares()...)
	}

	ButtonHandlers := []handler.ButtonHandler{
		handler.NewRemoveAllSubscriptionButton(appCore),
		handler.NewCancelRemoveAllSubscriptionButton(),
		handler.NewSetFeedItemButton(b.tb, appCore),
		handler.NewRemoveSubscriptionItemButton(appCore),
		handler.NewNotificationSwitchButton(b.tb, appCore),
		handler.NewSetSubscriptionTagButton(b.tb),
		handler.NewTelegraphSwitchButton(b.tb, appCore),
		handler.NewSubscriptionSwitchButton(b.tb, appCore),
	}

	for _, h := range ButtonHandlers {
		b.tb.Handle(h, h.Handle, h.Middlewares()...)
	}

	var commands []tb.Command
	for _, h := range commandHandlers {
		if h.Description() == "" {
			continue
		}
		commands = append(commands, tb.Command{Text: h.Command(), Description: h.Description()})
	}
	log.Debugf("set bot command %+v", commands)
	if err := b.tb.SetCommands(commands); err != nil {
		return err
	}
	return nil
}

func (b *Bot) Run() error {
	if config.RunMode == config.TestMode {
		return nil
	}

	if err := b.registerCommands(b.core); err != nil {
		return err
	}
	log.Infof("bot start %s", config.AppVersionInfo())
	b.tb.Start()
	return nil
}

func (b *Bot) SourceUpdate(
	source *model.Source, newContents []*model.Content, subscribes []*model.Subscribe,
) {
	b.BroadcastNews(source, subscribes, newContents)
}

func (b *Bot) SourceContentsEdit(
	source *model.Source, editedContents []*model.Content, subscribes []*model.Subscribe,
) {
	b.BroadcastEdit(source, subscribes, editedContents)
}

func (b *Bot) SourceUpdateError(source *model.Source) {
	b.BroadcastSourceError(source)
}

// renderContentMessage formats a content item for a specific subscriber
func (b *Bot) renderContentMessage(source *model.Source, sub *model.Subscribe, content *model.Content) (string, error) {
	previewLimit := sub.GetPreviewLimit(config.PreviewText)
	previewText := preview.TrimDescription(content.Description, previewLimit)
	contentTitle, subPreviewText := content.Title, previewText
	if sub.TranslateLang != "" {
		langName := translate.LanguageName(sub.TranslateLang)
		if b.translator == nil {
			zap.S().Warnw(
				"translation requested but translator not configured, pushing original",
				"user id", sub.UserID, "source id", sub.SourceID,
				"lang", sub.TranslateLang, "lang_name", langName,
			)
		} else {
			metaLang := content.Language
			if metaLang == "" && source != nil {
				metaLang = source.Language
			}
			zap.S().Debugw(
				"translating content for subscriber",
				"user id", sub.UserID,
				"source id", sub.SourceID,
				"hash", content.HashID,
				"lang", sub.TranslateLang,
				"lang_name", langName,
				"meta_lang", metaLang,
			)
			contentTitle, subPreviewText = b.translateContent(
				content.HashID, sub.TranslateLang, content.Title, previewText, metaLang,
			)
		}
	}

	publishedAt := ""
	if content.PublishedAt != nil {
		pubTime := *content.PublishedAt
		if sub.Timezone != "" {
			if loc, err := timezone.ParseLocation(sub.Timezone); err == nil && loc != nil {
				pubTime = pubTime.In(loc)
			}
		}
		publishedAt = pubTime.Format("2006-01-02 15:04:05 -07:00")
	}

	tpldata := &config.TplData{
		SourceTitle:     sub.DisplayTitle(source.Title),
		ContentTitle:    contentTitle,
		RawLink:         content.RawLink,
		PublishedAt:     publishedAt,
		PreviewText:     subPreviewText,
		TelegraphURL:    content.TelegraphURL,
		Tags:            sub.Tag,
		EnableTelegraph: sub.EnableTelegraph == 1 && content.TelegraphURL != "",
	}

	return tpldata.Render(config.MessageMode)
}

// BroadcastNews send new contents message to subscriber
func (b *Bot) BroadcastNews(source *model.Source, subs []*model.Subscribe, contents []*model.Content) {
	translatedSubs := 0
	for _, sub := range subs {
		if sub.TranslateLang != "" {
			translatedSubs++
		}
	}
	zap.S().Infow(
		"broadcast news",
		"fetcher id", source.ID,
		"fetcher title", source.Title,
		"subscriber count", len(subs),
		"translation enabled subscribers", translatedSubs,
		"new contents", len(contents),
	)

	for _, content := range contents {
		for _, sub := range subs {
			msg, err := b.renderContentMessage(source, sub, content)
			if err != nil {
				zap.S().Errorw(
					"broadcast news error, renderContentMessage err",
					"error", err.Error(),
				)
				continue
			}

			u := &tb.User{
				ID: sub.UserID,
			}
			o := &tb.SendOptions{
				DisableWebPagePreview: config.DisableWebPagePreview,
				ParseMode:             config.MessageMode,
				DisableNotification:   sub.EnableNotification != 1,
			}
			sentMsg, err := b.tb.Send(u, msg, o)
			if err != nil {
				if strings.Contains(err.Error(), "Forbidden") {
					zap.S().Errorw(
						"broadcast news error, bot stopped by user",
						"error", err.Error(),
						"user id", sub.UserID,
						"source id", sub.SourceID,
						"title", source.Title,
						"link", source.Link,
					)
					b.core.Unsubscribe(context.Background(), sub.UserID, sub.SourceID)
				}

				/*
					Telegram return error if markdown message has incomplete format.
					Print the msg to warn the user
					api error: Bad Request: can't parse entities: Can't find end of the entity starting at byte offset 894
				*/
				if strings.Contains(err.Error(), "parse entities") {
					zap.S().Errorw(
						"broadcast news error, markdown error",
						"markdown msg", msg,
						"error", err.Error(),
					)
				}
			} else if sentMsg != nil {
				if err := b.core.SaveContentMessage(context.Background(), content.HashID, sub.UserID, sentMsg.ID); err != nil {
					zap.S().Errorw(
						"save content message failed",
						"hash", content.HashID,
						"user id", sub.UserID,
						"message id", sentMsg.ID,
						"title", content.Title,
						"error", err.Error(),
					)
				} else {
					zap.S().Infow(
						"sent news message and recorded message ID",
						"user id", sub.UserID,
						"source id", sub.SourceID,
						"message id", sentMsg.ID,
						"hash", content.HashID,
						"title", content.Title,
					)
				}
			}
		}
	}
}

// BroadcastEdit edits previously sent messages for subscribers when content is updated
func (b *Bot) BroadcastEdit(source *model.Source, subs []*model.Subscribe, contents []*model.Content) {
	zap.S().Infow(
		"broadcast edit",
		"fetcher id", source.ID,
		"fetcher title", source.Title,
		"subscriber count", len(subs),
		"edited contents", len(contents),
	)

	for _, content := range contents {
		// NOTE: We intentionally do NOT delete translation cache here.
		// Our translation cache checks source text fingerprints (SrcTitleHash and SrcPreviewHash).
		// If only metadata (published time, link, telegraph url) was updated, the translation is safely
		// reused with 0 token consumption. If the text truly changed, translateContent detects it and
		// translates only the modified parts.

		for _, sub := range subs {
			contentMsg, err := b.core.GetContentMessage(context.Background(), content.HashID, sub.UserID)
			if err != nil || contentMsg == nil || contentMsg.MessageID == 0 {
				zap.S().Infow(
					"skip message edit: no prior message ID recorded for subscriber",
					"user id", sub.UserID,
					"source id", sub.SourceID,
					"hash", content.HashID,
					"title", content.Title,
				)
				continue
			}

			msg, err := b.renderContentMessage(source, sub, content)
			if err != nil {
				zap.S().Errorw(
					"broadcast edit error, renderContentMessage err",
					"error", err.Error(),
				)
				continue
			}

			targetMsg := &tb.Message{
				ID:   contentMsg.MessageID,
				Chat: &tb.Chat{ID: sub.UserID},
			}
			o := &tb.SendOptions{
				DisableWebPagePreview: config.DisableWebPagePreview,
				ParseMode:             config.MessageMode,
			}

			if _, err := b.tb.Edit(targetMsg, msg, o); err != nil {
				if strings.Contains(err.Error(), "message is not modified") {
					zap.S().Infow(
						"telegram message already up-to-date (not modified)",
						"user id", sub.UserID,
						"source id", sub.SourceID,
						"message id", contentMsg.MessageID,
						"hash", content.HashID,
						"title", content.Title,
					)
					continue
				}

				if strings.Contains(err.Error(), "Forbidden") {
					zap.S().Errorw(
						"broadcast edit error, bot stopped by user",
						"error", err.Error(),
						"user id", sub.UserID,
						"source id", sub.SourceID,
						"title", source.Title,
						"link", source.Link,
					)
					b.core.Unsubscribe(context.Background(), sub.UserID, sub.SourceID)
					continue
				}

				if strings.Contains(err.Error(), "parse entities") {
					zap.S().Errorw(
						"broadcast edit error, markdown error",
						"markdown msg", msg,
						"error", err.Error(),
					)
					continue
				}

				zap.S().Warnw(
					"telegram message edit failed",
					"user id", sub.UserID,
					"source id", sub.SourceID,
					"message id", contentMsg.MessageID,
					"hash", content.HashID,
					"title", content.Title,
					"error", err.Error(),
				)
			} else {
				zap.S().Infow(
					"telegram message edited successfully",
					"user id", sub.UserID,
					"source id", sub.SourceID,
					"message id", contentMsg.MessageID,
					"hash", content.HashID,
					"title", content.Title,
				)
			}
		}
	}
}

// translateContent returns the translated title and preview of one content in
// one target language, falling back to the original text when translation
// fails. Results are cached per (content hash, language) and validated against
// source text fingerprints so that content pushed to many subscribers or checked
// during polling updates triggers minimal LLM token consumption.
func (b *Bot) translateContent(hashID, lang, title, previewText string, metaLangs ...string) (string, string) {
	metaLang := ""
	if len(metaLangs) > 0 {
		metaLang = metaLangs[0]
	}
	titleHash := translate.HashText(title)
	previewHash := translate.HashText(previewText)
	key := fmt.Sprintf("%s|%s|%x", hashID, lang, previewHash)
	langName := translate.LanguageName(lang)

	cached, hasCache := b.transCache.Get(key)
	if hasCache && cached.SrcTitleHash == titleHash && cached.SrcPreviewHash == previewHash {
		zap.S().Debugw("translate cache hit (full)", "hash", hashID, "lang", lang, "lang_name", langName)
		return cached.Title, cached.Preview
	}

	titleNeedsTrans := strings.TrimSpace(title) != "" && !translate.IsSameLanguageWithMeta(title, lang, metaLang)
	previewNeedsTrans := strings.TrimSpace(previewText) != "" && !translate.IsSameLanguageWithMeta(previewText, lang, metaLang)

	if !titleNeedsTrans && !previewNeedsTrans {
		zap.S().Debugw(
			"translate skipped (already target language)",
			"hash", hashID,
			"lang", lang,
			"lang_name", langName,
			"meta_lang", metaLang,
		)
		b.transCache.Put(key, translate.CachedTranslation{
			SrcTitleHash:   titleHash,
			SrcPreviewHash: previewHash,
			Title:          title,
			Preview:        previewText,
		})
		return title, previewText
	}

	ctx := context.Background()
	zap.S().Debugw(
		"translate start",
		"hash", hashID,
		"lang", lang,
		"lang_name", langName,
		"meta_lang", metaLang,
		"title_needs", titleNeedsTrans,
		"preview_needs", previewNeedsTrans,
		"title_len", len([]rune(title)),
		"preview_len", len([]rune(previewText)),
	)

	var translatedTitle, translatedPreview string
	hasError := false

	titleUnchanged := hasCache && cached.SrcTitleHash == titleHash && cached.Title != ""
	previewUnchanged := hasCache && cached.SrcPreviewHash == previewHash && cached.Preview != ""

	if titleUnchanged && previewUnchanged {
		return cached.Title, cached.Preview
	}

	if titleUnchanged {
		translatedTitle = cached.Title
		if previewNeedsTrans {
			tPreview, err := b.translator.Translate(ctx, previewText, lang)
			if err != nil {
				hasError = true
				zap.S().Warnw("translate preview failed, fallback to original", "error", err.Error(), "lang", lang, "hash", hashID)
				translatedPreview = previewText
			} else {
				translatedPreview = tPreview
			}
		} else {
			translatedPreview = previewText
		}
	} else if previewUnchanged {
		translatedPreview = cached.Preview
		if titleNeedsTrans {
			tTitle, err := b.translator.Translate(ctx, title, lang)
			if err != nil {
				hasError = true
				zap.S().Warnw("translate title failed, fallback to original", "error", err.Error(), "lang", lang, "hash", hashID)
				translatedTitle = title
			} else {
				translatedTitle = tTitle
			}
		} else {
			translatedTitle = title
		}
	} else if !titleNeedsTrans {
		translatedTitle = title
		tPreview, err := b.translator.Translate(ctx, previewText, lang)
		if err != nil {
			hasError = true
			zap.S().Warnw("translate preview failed, fallback to original", "error", err.Error(), "lang", lang, "hash", hashID)
			translatedPreview = previewText
		} else {
			translatedPreview = tPreview
		}
	} else if !previewNeedsTrans {
		translatedPreview = previewText
		tTitle, err := b.translator.Translate(ctx, title, lang)
		if err != nil {
			hasError = true
			zap.S().Warnw("translate title failed, fallback to original", "error", err.Error(), "lang", lang, "hash", hashID)
			translatedTitle = title
		} else {
			translatedTitle = tTitle
		}
	} else {
		// Both changed or initial fetch -> combined translation in a single request
		tTitle, tPreview, err := b.translator.TranslateContent(ctx, title, previewText, lang)
		if err != nil {
			hasError = true
			zap.S().Warnw("translate content failed, fallback to original", "error", err.Error(), "lang", lang, "hash", hashID)
			translatedTitle = title
			translatedPreview = previewText
		} else {
			translatedTitle = tTitle
			translatedPreview = tPreview
		}
	}

	if !hasError {
		b.transCache.Put(key, translate.CachedTranslation{
			SrcTitleHash:   titleHash,
			SrcPreviewHash: previewHash,
			Title:          translatedTitle,
			Preview:        translatedPreview,
		})
	}
	return translatedTitle, translatedPreview
}

// truncate limits a string to n runes for log readability.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}

// BroadcastSourceError send fetcher update error message to subscribers
func (b *Bot) BroadcastSourceError(source *model.Source) {
	subs, err := b.core.GetSourceAllSubscriptions(context.Background(), source.ID)
	if err != nil {
		log.Errorf("get subscriptions failed, %v", err)
	}
	var u tb.User
	for _, sub := range subs {
		message := fmt.Sprintf(
			"[%s](%s) 已经累计连续%d次更新失败，暂停更新",
			sub.DisplayTitle(source.Title), source.Link, config.ErrorThreshold,
		)
		u.ID = sub.UserID
		_, _ = b.tb.Send(
			&u, message, &tb.SendOptions{
				ParseMode: tb.ModeMarkdown,
			},
		)
	}
}
