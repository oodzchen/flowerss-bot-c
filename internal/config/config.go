package config

import (
	"fmt"
	"text/template"

	"github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
	tb "gopkg.in/telebot.v3"
)

type RunType string

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"

	ProjectName          string = "flowerss"
	BotToken             string
	Socks5               string
	TelegraphToken       []string
	TelegraphAccountName string
	TelegraphAuthorName  string = "flowerss-bot"
	TelegraphAuthorURL   string

	// EnableTelegraph 是否启用telegraph
	EnableTelegraph       bool = false
	PreviewText           int  = 300
	DisableWebPagePreview bool = false
	mysqlConfig           *mysql.Config
	SQLitePath            string
	EnableMysql           bool = false

	// UpdateInterval rss抓取间隔
	UpdateInterval int = 10

	// ErrorThreshold rss源抓取错误阈值
	ErrorThreshold uint = 100

	// MessageTpl rss更新推送模版
	MessageTpl *template.Template

	// MessageMode telegram消息渲染模式
	MessageMode tb.ParseMode

	// TelegramEndpoint telegram bot 服务器地址，默认为空
	TelegramEndpoint string = tb.DefaultApiURL

	// UserAgent User-Agent
	UserAgent string

	// RunMode 运行模式 Release / Debug
	RunMode RunType = ReleaseMode

	// AllowUsers 允许使用bot的用户
	AllowUsers []int64

	// DBLogMode 是否打印数据库日志
	DBLogMode bool = false

	// TranslateProvider 翻译服务提供商（llm/openrouter = OpenAI 兼容接口，后续可扩展 google/deepl 等）
	TranslateProvider string = "llm"
	// TranslateBaseURL OpenAI 兼容翻译接口地址
	TranslateBaseURL string = "https://api.deepseek.com/v1"
	// TranslateAPIKey 翻译接口 API Key（Ollama 等本地服务可留空）
	TranslateAPIKey string
	// TranslateModel 翻译使用的模型名；OpenRouter 需填 厂商/模型 形式的模型 ID
	TranslateModel string = "deepseek-chat"
	// TranslateHTTPReferer OpenRouter 可选：你的站点地址，用于排行榜统计
	TranslateHTTPReferer string
	// TranslateXTitle OpenRouter 可选：应用名称，用于排行榜统计
	TranslateXTitle string
)

const (
	defaultMessageTplMode = tb.ModeHTML
	defaultMessageTpl     = `<b>{{.SourceTitle}}</b>
{{if .EnableTelegraph}}<a href="{{.TelegraphURL}}">{{.ContentTitle}}</a> | <a href="{{.RawLink}}">原文</a>
{{else}}<a href="{{.RawLink}}">{{.ContentTitle}}</a>
{{end}}{{if .PublishedAt}}发布时间：{{.PublishedAt}}
{{end}}{{if .PreviewText}}
{{.PreviewText}}
{{end}}{{if .Tags}}{{.Tags}}
{{end}}
`
	defaultMessageMarkdownTpl = `*{{.SourceTitle}}*
{{if .EnableTelegraph}}[{{.ContentTitle}}]({{.TelegraphURL}}) | [原文]({{.RawLink}})
{{else}}[{{.ContentTitle}}]({{.RawLink}})
{{end}}{{if .PublishedAt}}发布时间：{{.PublishedAt}}
{{end}}{{if .PreviewText}}
{{.PreviewText}}
{{end}}{{if .Tags}}{{.Tags}}
{{end}}
`
	TestMode    RunType = "Test"
	ReleaseMode RunType = "Release"
)

type TplData struct {
	SourceTitle     string
	ContentTitle    string
	RawLink         string
	PublishedAt     string
	PreviewText     string
	TelegraphURL    string
	Tags            string
	EnableTelegraph bool
}

func AppVersionInfo() (s string) {
	s = fmt.Sprintf("version %v, commit %v, built at %v", version, commit, date)
	return
}

// GetString get string config value by key
func GetString(key string) string {
	var value string
	if viper.IsSet(key) {
		value = viper.GetString(key)
	}

	return value
}

func GetMysqlDSN() string {
	return mysqlConfig.FormatDSN()
}
