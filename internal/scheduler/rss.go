package scheduler

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
	"go.uber.org/atomic"

	"github.com/indes/flowerss-bot/internal/config"
	"github.com/indes/flowerss-bot/internal/core"
	"github.com/indes/flowerss-bot/internal/feed"
	"github.com/indes/flowerss-bot/internal/log"
	"github.com/indes/flowerss-bot/internal/model"
	tgraph "github.com/indes/flowerss-bot/internal/preview"
	"github.com/indes/flowerss-bot/pkg/client"
)

// RssUpdateObserver Rss Update observer
type RssUpdateObserver interface {
	SourceUpdate(*model.Source, []*model.Content, []*model.Subscribe)
	SourceContentsEdit(*model.Source, []*model.Content, []*model.Subscribe)
	SourceUpdateError(*model.Source)
}

// NewRssTask new RssUpdateTask
func NewRssTask(appCore *core.Core) *RssUpdateTask {
	return &RssUpdateTask{
		observerList: []RssUpdateObserver{},
		core:         appCore,
		feedParser:   appCore.FeedParser(),
		httpClient:   appCore.HttpClient(),
	}
}

// RssUpdateTask rss更新任务
type RssUpdateTask struct {
	observerList []RssUpdateObserver
	isStop       atomic.Bool
	core         *core.Core
	feedParser   *feed.FeedParser
	httpClient   *client.HttpClient
}

type contentUpdate struct {
	source  *model.Source
	content *model.Content
	subs    []*model.Subscribe
	isEdit  bool
}

func isTimeDifferent(t1, t2 *time.Time) bool {
	if t1 == nil && t2 == nil {
		return false
	}
	if t1 == nil || t2 == nil {
		return true
	}
	return !t1.Equal(*t2)
}

func isContentEdited(old *model.Content, item *gofeed.Item) bool {
	newTitle := strings.Trim(item.Title, " ")
	newDesc := feed.ItemDescription(item)
	newLink := item.Link
	newPub := feed.ItemPublishedAt(item)

	if old.Title != newTitle || old.RawLink != newLink || isTimeDifferent(old.PublishedAt, newPub) {
		return true
	}

	if old.Description != "" && old.Description != newDesc {
		return true
	}

	return false
}

func sortContentUpdatesOldestFirst(updates []contentUpdate) {
	sort.SliceStable(updates, func(i, j int) bool {
		left := updates[i].content.PublishedAt
		right := updates[j].content.PublishedAt
		switch {
		case left == nil && right == nil:
			return false
		case left == nil:
			return true
		case right == nil:
			return false
		default:
			return left.Before(*right)
		}
	})
}

// Register 注册rss更新订阅者
func (t *RssUpdateTask) Register(observer RssUpdateObserver) {
	t.observerList = append(t.observerList, observer)
}

// Stop scheduler
func (t *RssUpdateTask) Stop() {
	t.isStop.Store(true)
}

// Start run scheduler
func (t *RssUpdateTask) Start() {
	if config.RunMode == config.TestMode {
		return
	}

	t.isStop.Store(false)
	go func() {
		for {
			if t.isStop.Load() {
				log.Info("RssUpdateTask stopped")
				return
			}

			sources, err := t.core.GetSources(context.Background())
			if err != nil {
				log.Errorf("get sources failed, %v", err)
				time.Sleep(time.Duration(config.UpdateInterval) * time.Minute)
				continue
			}
			if len(sources) > 0 {
				log.Infof("start updating %d sources", len(sources))
			}
			var updates []contentUpdate
			for _, source := range sources {
				if source.ErrorCount >= config.ErrorThreshold {
					continue
				}

				newContents, editContents, err := t.getSourceContents(source)
				if err != nil {
					if source.ErrorCount >= config.ErrorThreshold {
						t.notifyAllObserverErrorUpdate(source)
					}
					continue
				}

				if len(newContents) > 0 || len(editContents) > 0 {
					subs, err := t.core.GetSourceAllSubscriptions(
						context.Background(), source.ID,
					)
					if err != nil {
						log.Errorf("get subscriptions failed, %v", err)
						continue
					}
					for _, content := range newContents {
						updates = append(updates, contentUpdate{source: source, content: content, subs: subs, isEdit: false})
					}
					for _, content := range editContents {
						updates = append(updates, contentUpdate{source: source, content: content, subs: subs, isEdit: true})
					}
				}
			}

			// Feed documents commonly list newest entries first. Queue every new
			// and edited item from this scan and publish chronologically across all sources.
			sortContentUpdatesOldestFirst(updates)
			for _, update := range updates {
				if update.isEdit {
					t.notifyAllObserverEdit(update.source, []*model.Content{update.content}, update.subs)
				} else {
					t.notifyAllObserverUpdate(update.source, []*model.Content{update.content}, update.subs)
				}
			}

			time.Sleep(time.Duration(config.UpdateInterval) * time.Minute)
		}
	}()
}

// getSourceContents 获取rss内容（包括新文章和被编辑的文章）
func (t *RssUpdateTask) getSourceContents(source *model.Source) ([]*model.Content, []*model.Content, error) {
	log.Infof("fetching source [%d] %s", source.ID, source.Link)

	rssFeed, err := t.feedParser.ParseFromURL(context.Background(), source.Link)
	if err != nil {
		log.Errorf("fetch source [%d] %s failed, err: %v", source.ID, source.Link, err)
		t.core.SourceErrorCountIncr(context.Background(), source.ID)
		return nil, nil, err
	}
	t.core.ClearSourceErrorCount(context.Background(), source.ID)

	feedLang := feed.ItemLanguage(rssFeed, nil)
	if source.Language == "" && feedLang != "" {
		source.Language = feedLang
	}

	newContents, editContents, err := t.processFeedItems(source, rssFeed.Items, feedLang)
	if err != nil {
		log.Errorf("process contents for source [%d] %s failed, err: %v", source.ID, source.Link, err)
		return nil, nil, err
	}
	log.Infof("fetch source [%d] %s success, %d items fetched, %d new contents, %d edited contents",
		source.ID, source.Link, len(rssFeed.Items), len(newContents), len(editContents))
	return newContents, editContents, nil
}

// processFeedItems handles feed items by separating new items and edited items
func (t *RssUpdateTask) processFeedItems(
	s *model.Source, items []*gofeed.Item, feedLangs ...string,
) ([]*model.Content, []*model.Content, error) {
	feedLang := ""
	if len(feedLangs) > 0 {
		feedLang = feedLangs[0]
	}
	if feedLang == "" && s != nil {
		feedLang = s.Language
	}
	var newItems []*gofeed.Item
	var editContents []*model.Content

	for _, item := range feed.SortItemsOldestFirst(items) {
		itemGUID := feed.ItemGUID(item)
		hashID := model.GenHashID(s.Link, itemGUID)

		oldContent, err := t.core.GetContent(context.Background(), hashID)
		if err != nil || oldContent == nil {
			newItems = append(newItems, item)
			continue
		}

		if isContentEdited(oldContent, item) {
			previewURL := oldContent.TelegraphURL
			itemContent := feed.ItemContent(item)
			if config.EnableTelegraph && len([]rune(itemContent)) > config.PreviewText {
				if newURL, err := tgraph.PublishHtml(s.Title, item.Title, item.Link, itemContent); err == nil && newURL != "" {
					previewURL = newURL
				}
			}

			itemLang := feed.ItemLanguage(nil, item)
			if itemLang == "" {
				itemLang = feedLang
			}
			if itemLang != "" {
				oldContent.Language = itemLang
			}

			oldContent.Title = strings.Trim(item.Title, " ")
			oldContent.Description = feed.ItemDescription(item)
			oldContent.RawLink = item.Link
			oldContent.TelegraphURL = previewURL
			oldContent.PublishedAt = feed.ItemPublishedAt(item)

			if err := t.core.UpdateContent(context.Background(), hashID, oldContent); err != nil {
				log.Errorf("update content %s failed: %v", hashID, err)
			} else {
				editContents = append(editContents, oldContent)
			}
		} else if oldContent.Description == "" && feed.ItemDescription(item) != "" {
			oldContent.Description = feed.ItemDescription(item)
			itemLang := feed.ItemLanguage(nil, item)
			if itemLang == "" {
				itemLang = feedLang
			}
			if itemLang != "" {
				oldContent.Language = itemLang
			}
			_ = t.core.UpdateContent(context.Background(), hashID, oldContent)
		}
	}

	newContents, err := t.core.AddSourceContents(context.Background(), s, newItems)
	if err != nil {
		return nil, nil, err
	}
	return newContents, editContents, nil
}

// notifyAllObserverUpdate notify all rss SourceUpdate observer
func (t *RssUpdateTask) notifyAllObserverUpdate(
	source *model.Source, newContents []*model.Content, subscribes []*model.Subscribe,
) {
	wg := sync.WaitGroup{}
	for _, observer := range t.observerList {
		wg.Add(1)
		go func(o RssUpdateObserver) {
			defer wg.Done()
			o.SourceUpdate(source, newContents, subscribes)
		}(observer)
	}
	wg.Wait()
}

// notifyAllObserverEdit notify all rss SourceContentsEdit observer
func (t *RssUpdateTask) notifyAllObserverEdit(
	source *model.Source, editedContents []*model.Content, subscribes []*model.Subscribe,
) {
	wg := sync.WaitGroup{}
	for _, observer := range t.observerList {
		wg.Add(1)
		go func(o RssUpdateObserver) {
			defer wg.Done()
			o.SourceContentsEdit(source, editedContents, subscribes)
		}(observer)
	}
	wg.Wait()
}

// notifyAllObserverErrorUpdate notify all rss error SourceUpdate observer
func (t *RssUpdateTask) notifyAllObserverErrorUpdate(source *model.Source) {
	wg := sync.WaitGroup{}
	for _, observer := range t.observerList {
		wg.Add(1)
		go func(o RssUpdateObserver) {
			defer wg.Done()
			o.SourceUpdateError(source)
		}(observer)
	}
	wg.Wait()
}

