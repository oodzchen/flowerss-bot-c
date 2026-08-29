package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/indes/flowerss-bot/internal/config"
	"github.com/indes/flowerss-bot/internal/feed"
	"github.com/indes/flowerss-bot/internal/log"
	"github.com/indes/flowerss-bot/internal/model"
	"github.com/indes/flowerss-bot/internal/preview"
	"github.com/indes/flowerss-bot/internal/storage"
	"github.com/indes/flowerss-bot/pkg/client"
)

var (
	ErrSubscriptionExist    = errors.New("already subscribed")
	ErrSubscriptionNotExist = errors.New("subscription not exist")
	ErrSourceNotExist       = errors.New("source not exist")
	ErrContentNotExist      = errors.New("content not exist")
)

type Core struct {
	// Storage
	userStorage         storage.User
	contentStorage      storage.Content
	sourceStorage       storage.Source
	subscriptionStorage storage.Subscription

	feedParser *feed.FeedParser
	httpClient *client.HttpClient
}

func (c *Core) FeedParser() *feed.FeedParser {
	return c.feedParser
}

func (c *Core) HttpClient() *client.HttpClient {
	return c.httpClient
}

func NewCore(
	userStorage storage.User,
	contentStorage storage.Content,
	sourceStorage storage.Source,
	subscriptionStorage storage.Subscription,
	parser *feed.FeedParser,
	httpClient *client.HttpClient,
) *Core {
	return &Core{
		userStorage:         userStorage,
		contentStorage:      contentStorage,
		sourceStorage:       sourceStorage,
		subscriptionStorage: subscriptionStorage,
		feedParser:          parser,
		httpClient:          httpClient,
	}
}

func NewCoreFormConfig() *Core {
	var err error
	var db *gorm.DB
	if config.EnableMysql {
		db, err = gorm.Open(mysql.Open(config.GetMysqlDSN()))
	} else {
		db, err = gorm.Open(sqlite.Open(config.SQLitePath))
	}
	if err != nil {
		log.Fatalf("connect db failed, err: %+v", err)
		return nil
	}

	if config.DBLogMode {
		db = db.Debug()
	}

	sqlDB, err := db.DB()
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)

	subscriptionStorage := storage.NewSubscriptionStorageImpl(db)

	// httpclient
	clientOpts := []client.HttpClientOption{
		client.WithTimeout(10 * time.Second),
	}
	if config.Socks5 != "" {
		clientOpts = append(clientOpts, client.WithProxyURL(fmt.Sprintf("socks5://%s", config.Socks5)))
	}

	if config.UserAgent != "" {
		clientOpts = append(clientOpts, client.WithUserAgent(config.UserAgent))
	}
	httpClient := client.NewHttpClient(clientOpts...)

	// feedParser
	feedParser := feed.NewFeedParser(httpClient)

	return NewCore(
		storage.NewUserStorageImpl(db),
		storage.NewContentStorageImpl(db),
		storage.NewSourceStorageImpl(db),
		subscriptionStorage,
		feedParser,
		httpClient,
	)
}

func (c *Core) Init() error {
	if err := c.userStorage.Init(context.Background()); err != nil {
		return err
	}
	if err := c.contentStorage.Init(context.Background()); err != nil {
		return err
	}
	if err := c.sourceStorage.Init(context.Background()); err != nil {
		return err
	}
	if err := c.subscriptionStorage.Init(context.Background()); err != nil {
		return err
	}
	return nil
}

// GetUserSubscribedSources 获取用户订阅的订阅源
func (c *Core) GetUserSubscribedSources(ctx context.Context, userID int64) ([]*model.Source, error) {
	opt := &storage.GetSubscriptionsOptions{Count: -1}
	result, err := c.subscriptionStorage.GetSubscriptionsByUserID(ctx, userID, opt)
	if err != nil {
		return nil, err
	}

	var sources []*model.Source
	for _, subs := range result.Subscriptions {
		source, err := c.sourceStorage.GetSource(ctx, subs.SourceID)
		if err != nil {
			log.Errorf("get source %d failed, %v", subs.SourceID, err)
			continue
		}
		source.Title = subs.DisplayTitle(source.Title)
		sources = append(sources, source)
	}
	return sources, nil
}

// AddSubscription 添加订阅
func (c *Core) AddSubscription(ctx context.Context, userID int64, sourceID uint) error {
	exist, err := c.subscriptionStorage.SubscriptionExist(ctx, userID, sourceID)
	if err != nil {
		return err
	}

	if exist {
		return ErrSubscriptionExist
	}

	subscription := &model.Subscribe{
		UserID:             userID,
		SourceID:           sourceID,
		EnableNotification: 1,
		EnableTelegraph:    1,
		Interval:           config.UpdateInterval,
		WaitTime:           config.UpdateInterval,
	}
	return c.subscriptionStorage.AddSubscription(ctx, subscription)
}

// Unsubscribe 添加订阅
func (c *Core) Unsubscribe(ctx context.Context, userID int64, sourceID uint) error {
	exist, err := c.subscriptionStorage.SubscriptionExist(ctx, userID, sourceID)
	if err != nil {
		return err
	}

	if !exist {
		return ErrSubscriptionNotExist
	}

	// 移除该用户订阅
	_, err = c.subscriptionStorage.DeleteSubscription(ctx, userID, sourceID)
	if err != nil {
		return err
	}

	// 获取源的订阅数量
	count, err := c.subscriptionStorage.CountSourceSubscriptions(ctx, sourceID)
	if err != nil {
		return err
	}

	if count != 0 {
		return nil
	}

	// 如果源不再有订阅用户，移除该订阅源
	if err := c.removeSource(ctx, sourceID); err != nil {
		return err
	}
	return nil
}

// removeSource 移除订阅源
func (c *Core) removeSource(ctx context.Context, sourceID uint) error {
	if err := c.sourceStorage.Delete(ctx, sourceID); err != nil {
		return err
	}

	count, err := c.contentStorage.DeleteSourceContents(ctx, sourceID)
	if err != nil {
		return err
	}
	log.Infof("remove source %d and %d contents", sourceID, count)
	return nil
}

// GetSourceByURL 获取用户订阅的订阅源
func (c *Core) GetSourceByURL(ctx context.Context, sourceURL string) (*model.Source, error) {
	source, err := c.sourceStorage.GetSourceByURL(ctx, sourceURL)
	if err != nil {
		if err == storage.ErrRecordNotFound {
			return nil, ErrSourceNotExist
		}
		return nil, err
	}
	return source, nil
}

// GetSource 获取用户订阅的订阅源
func (c *Core) GetSource(ctx context.Context, id uint) (*model.Source, error) {
	source, err := c.sourceStorage.GetSource(ctx, id)
	if err != nil {
		if err == storage.ErrRecordNotFound {
			return nil, ErrSourceNotExist
		}
		return nil, err
	}
	return source, nil
}

// GetSource 获取用户订阅的订阅源
func (c *Core) GetSources(ctx context.Context) ([]*model.Source, error) {
	return c.sourceStorage.GetSources(ctx)
}

// CreateSource 创建订阅源
func (c *Core) CreateSource(ctx context.Context, sourceURL string) (*model.Source, error) {
	s, err := c.GetSourceByURL(ctx, sourceURL)
	if err == nil {
		return s, nil
	}

	if err != nil && err != ErrSourceNotExist {
		return nil, err
	}

	log.Infof("fetching source %s", sourceURL)
	rssFeed, err := c.feedParser.ParseFromURL(ctx, sourceURL)
	if err != nil {
		log.Errorf("fetch source %s failed, %v", sourceURL, err)
		return nil, err
	}
	log.Infof("fetch source %s success, title: %s, %d items fetched", sourceURL, rssFeed.Title, len(rssFeed.Items))

	s = &model.Source{
		Title:      rssFeed.Title,
		Link:       sourceURL,
		Language:   feed.ItemLanguage(rssFeed, nil),
		ErrorCount: config.ErrorThreshold + 1, // 避免task更新
	}

	if err := c.sourceStorage.AddSource(ctx, s); err != nil {
		log.Errorf("add source failed, %v", err)
		return nil, err
	}
	defer c.ClearSourceErrorCount(ctx, s.ID)

	if _, err := c.AddSourceContents(ctx, s, rssFeed.Items); err != nil {
		log.Errorf("add source content failed, %v", err)
		return nil, err
	}
	return s, nil
}

func (c *Core) AddSourceContents(
	ctx context.Context, source *model.Source, items []*gofeed.Item,
) ([]*model.Content, error) {
	var wg sync.WaitGroup
	var contents []*model.Content
	for _, item := range feed.SortItemsOldestFirst(items) {
		wg.Add(1)
		previewURL := ""
		itemContent := feed.ItemContent(item)
		previewThreshold := config.PreviewText
		if previewThreshold < 0 {
			previewThreshold = -previewThreshold
		}
		if config.EnableTelegraph && previewThreshold > 0 && len([]rune(itemContent)) > previewThreshold {
			previewURL, _ = tgraph.PublishHtml(source.Title, item.Title, item.Link, itemContent)
		}
		itemGUID := feed.ItemGUID(item)
		itemLang := feed.ItemLanguage(nil, item)
		if itemLang == "" && source != nil && source.Language != "" {
			itemLang = source.Language
		}
		content := &model.Content{
			Title:        strings.Trim(item.Title, " "),
			Description:  feed.ItemDescription(item),
			SourceID:     source.ID,
			RawID:        itemGUID,
			HashID:       model.GenHashID(source.Link, itemGUID),
			RawLink:      item.Link,
			TelegraphURL: previewURL,
			PublishedAt:  feed.ItemPublishedAt(item),
			Language:     itemLang,
		}
		contents = append(contents, content)
		go func() {
			defer wg.Done()
			if err := c.contentStorage.AddContent(ctx, content); err != nil {
				log.Errorf("add content %#v failed, %v", content, err)
			}
		}()
	}
	wg.Wait()
	return contents, nil
}

// UnsubscribeAllSource 添加订阅
func (c *Core) UnsubscribeAllSource(ctx context.Context, userID int64) error {
	sources, err := c.GetUserSubscribedSources(ctx, userID)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for i := range sources {
		wg.Add(1)
		i := i
		go func() {
			defer wg.Done()
			if err := c.Unsubscribe(ctx, userID, sources[i].ID); err != nil {
				log.Errorf("user %d unsubscribe %d failed, %v", userID, sources[i].ID, err)
			}
		}()
	}
	wg.Wait()
	return nil
}

// GetSubscription 获取订阅
func (c *Core) GetSubscription(ctx context.Context, userID int64, sourceID uint) (*model.Subscribe, error) {
	subscription, err := c.subscriptionStorage.GetSubscription(ctx, userID, sourceID)
	if err != nil {
		if err == storage.ErrRecordNotFound {
			return nil, ErrSubscriptionNotExist
		}
		return nil, err
	}
	return subscription, nil
}

// SetSubscriptionTag 设置订阅标签
func (c *Core) SetSubscriptionTag(ctx context.Context, userID int64, sourceID uint, tags []string) error {
	subscription, err := c.GetSubscription(ctx, userID, sourceID)
	if err != nil {
		return err
	}

	subscription.Tag = "#" + strings.Join(tags, " #")
	return c.subscriptionStorage.UpdateSubscription(ctx, userID, sourceID, subscription)
}

// SetSubscriptionTitle sets a title used only by this subscription. An empty
// title restores the title supplied by the RSS source.
func (c *Core) SetSubscriptionTitle(ctx context.Context, userID int64, sourceID uint, title string) error {
	subscription, err := c.GetSubscription(ctx, userID, sourceID)
	if err != nil {
		return err
	}

	subscription.Title = strings.TrimSpace(title)
	// Save the whole subscription so an empty title is persisted when the user
	// restores the RSS source title. GORM Updates(struct) skips zero values.
	return c.subscriptionStorage.UpsertSubscription(ctx, userID, sourceID, subscription)
}

// SetSubscriptionLang sets the translation target language of one subscription.
// An empty lang disables translation for it.
func (c *Core) SetSubscriptionLang(ctx context.Context, userID int64, sourceID uint, lang string) error {
	if _, err := c.GetSubscription(ctx, userID, sourceID); err != nil {
		return err
	}

	// 显式列更新，避免依赖 GORM Save 的版本相关分支语义
	_, err := c.subscriptionStorage.UpdateSubscriptionLang(ctx, userID, sourceID, lang)
	return err
}

// SetSubscriptionLangForAll sets the translation target language of every
// subscription owned by userID, returning how many subscriptions changed.
func (c *Core) SetSubscriptionLangForAll(ctx context.Context, userID int64, lang string) (int, error) {
	count, err := c.subscriptionStorage.UpdateSubscriptionsLang(ctx, userID, lang)
	return int(count), err
}

// SetSubscriptionTimezone sets the timezone of one subscription.
// An empty tz resets timezone for it.
func (c *Core) SetSubscriptionTimezone(ctx context.Context, userID int64, sourceID uint, tz string) error {
	if _, err := c.GetSubscription(ctx, userID, sourceID); err != nil {
		return err
	}

	_, err := c.subscriptionStorage.UpdateSubscriptionTimezone(ctx, userID, sourceID, tz)
	return err
}

// SetSubscriptionTimezoneForAll sets the timezone of every subscription owned by userID.
func (c *Core) SetSubscriptionTimezoneForAll(ctx context.Context, userID int64, tz string) (int, error) {
	count, err := c.subscriptionStorage.UpdateSubscriptionsTimezone(ctx, userID, tz)
	return int(count), err
}

// SetSubscriptionPreviewLength sets the preview text length limit of one subscription.
// A nil length resets it to the global configuration.
func (c *Core) SetSubscriptionPreviewLength(ctx context.Context, userID int64, sourceID uint, length *int) error {
	if _, err := c.GetSubscription(ctx, userID, sourceID); err != nil {
		return err
	}

	_, err := c.subscriptionStorage.UpdateSubscriptionPreviewLength(ctx, userID, sourceID, length)
	return err
}

// SetSubscriptionPreviewLengthForAll sets the preview text length limit of every subscription owned by userID.
func (c *Core) SetSubscriptionPreviewLengthForAll(ctx context.Context, userID int64, length *int) (int, error) {
	count, err := c.subscriptionStorage.UpdateSubscriptionsPreviewLength(ctx, userID, length)
	return int(count), err
}

// SetSubscriptionInterval
func (c *Core) SetSubscriptionInterval(ctx context.Context, userID int64, sourceID uint, interval int) error {
	subscription, err := c.GetSubscription(ctx, userID, sourceID)
	if err != nil {
		return err
	}

	subscription.Interval = interval
	return c.subscriptionStorage.UpdateSubscription(ctx, userID, sourceID, subscription)
}

// EnableSourceUpdate 开启订阅源更新
func (c *Core) EnableSourceUpdate(ctx context.Context, sourceID uint) error {
	return c.ClearSourceErrorCount(ctx, sourceID)
}

// DisableSourceUpdate 关闭订阅源更新
func (c *Core) DisableSourceUpdate(ctx context.Context, sourceID uint) error {
	source, err := c.GetSource(ctx, sourceID)
	if err != nil {
		return err
	}

	source.ErrorCount = config.ErrorThreshold + 1
	return c.sourceStorage.UpsertSource(ctx, sourceID, source)
}

// ClearSourceErrorCount 清空订阅源错误计数
func (c *Core) ClearSourceErrorCount(ctx context.Context, sourceID uint) error {
	source, err := c.GetSource(ctx, sourceID)
	if err != nil {
		return err
	}

	source.ErrorCount = 0
	return c.sourceStorage.UpsertSource(ctx, sourceID, source)
}

// SourceErrorCountIncr 增加订阅源错误计数
func (c *Core) SourceErrorCountIncr(ctx context.Context, sourceID uint) error {
	source, err := c.GetSource(ctx, sourceID)
	if err != nil {
		return err
	}

	source.ErrorCount += 1
	return c.sourceStorage.UpsertSource(ctx, sourceID, source)
}

func (c *Core) ToggleSubscriptionNotice(ctx context.Context, userID int64, sourceID uint) error {
	subscription, err := c.GetSubscription(ctx, userID, sourceID)
	if err != nil {
		return err
	}
	if subscription.EnableNotification == 1 {
		subscription.EnableNotification = 0
	} else {
		subscription.EnableNotification = 1
	}
	return c.subscriptionStorage.UpsertSubscription(ctx, userID, sourceID, subscription)
}

func (c *Core) ToggleSourceUpdateStatus(ctx context.Context, sourceID uint) error {
	source, err := c.GetSource(ctx, sourceID)
	if err != nil {
		return err
	}

	if source.ErrorCount < config.ErrorThreshold {
		source.ErrorCount = config.ErrorThreshold + 1
	} else {
		source.ErrorCount = 0
	}
	return c.sourceStorage.UpsertSource(ctx, sourceID, source)
}

func (c *Core) ToggleSubscriptionTelegraph(ctx context.Context, userID int64, sourceID uint) error {
	subscription, err := c.GetSubscription(ctx, userID, sourceID)
	if err != nil {
		return err
	}
	if subscription.EnableTelegraph == 1 {
		subscription.EnableTelegraph = 0
	} else {
		subscription.EnableTelegraph = 1
	}
	return c.subscriptionStorage.UpsertSubscription(ctx, userID, sourceID, subscription)
}

func (c *Core) GetSourceAllSubscriptions(
	ctx context.Context, sourceID uint,
) ([]*model.Subscribe, error) {
	opt := &storage.GetSubscriptionsOptions{
		Count: -1,
	}
	result, err := c.subscriptionStorage.GetSubscriptionsBySourceID(ctx, sourceID, opt)
	if err != nil {
		return nil, err
	}
	return result.Subscriptions, nil
}

func (c *Core) ContentHashIDExist(
	ctx context.Context, hashID string,
) (bool, error) {
	result, err := c.contentStorage.HashIDExist(ctx, hashID)
	if err != nil {
		return false, err
	}
	return result, nil
}

func (c *Core) GetContent(
	ctx context.Context, hashID string,
) (*model.Content, error) {
	content, err := c.contentStorage.GetContent(ctx, hashID)
	if err != nil {
		if errors.Is(err, storage.ErrRecordNotFound) {
			return nil, ErrContentNotExist
		}
		return nil, err
	}
	return content, nil
}

func (c *Core) UpdateContent(
	ctx context.Context, hashID string, newContent *model.Content,
) error {
	return c.contentStorage.UpdateContent(ctx, hashID, newContent)
}

func (c *Core) SaveContentMessage(
	ctx context.Context, hashID string, userID int64, messageID int,
) error {
	msg := &model.ContentMessage{
		HashID:    hashID,
		UserID:    userID,
		MessageID: messageID,
	}
	return c.contentStorage.SaveContentMessage(ctx, msg)
}

func (c *Core) GetContentMessage(
	ctx context.Context, hashID string, userID int64,
) (*model.ContentMessage, error) {
	return c.contentStorage.GetContentMessage(ctx, hashID, userID)
}

func (c *Core) GetContentMessages(
	ctx context.Context, hashID string,
) ([]*model.ContentMessage, error) {
	return c.contentStorage.GetContentMessages(ctx, hashID)
}

