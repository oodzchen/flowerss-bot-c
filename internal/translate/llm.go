package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/indes/flowerss-bot/pkg/client"
)

// LLMTranslator translates text through any OpenAI-compatible chat completions
// endpoint (DeepSeek, OpenAI, OpenRouter, Ollama, GLM, Moonshot, ...).
type LLMTranslator struct {
	httpClient  *client.HttpClient
	baseURL     string
	apiKey      string
	model       string
	httpReferer string // OpenRouter 可选：HTTP-Referer 头
	xTitle      string // OpenRouter 可选：X-Title 头
}

func NewLLMTranslator(httpClient *client.HttpClient, baseURL, apiKey, model string) *LLMTranslator {
	return &LLMTranslator{
		httpClient: httpClient,
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
	}
}

// SetHTTPReferer sets the optional HTTP-Referer header recommended by
// OpenRouter for ranking and analytics.
func (t *LLMTranslator) SetHTTPReferer(referer string) {
	t.httpReferer = referer
}

// SetXTitle sets the optional X-Title header recommended by OpenRouter for
// ranking and analytics.
func (t *LLMTranslator) SetXTitle(title string) {
	t.xTitle = title
}

const systemPrompt = `You are a professional translation engine. Translate the user's text into the requested language. Requirements:
1. Output ONLY the translation, no explanations, no quotes, no code blocks, no extra words.
2. Preserve URLs, links, numbers, product names and proper nouns where appropriate.
3. Preserve the original line breaks and paragraph structure.
4. Keep the original tone (formal or casual) and style.
5. If the text is already in the requested language, return it unchanged.`

const combinedSystemPrompt = `You are a professional translation engine. Translate the user's Title and Preview text into the requested language. Requirements:
1. Output ONLY the translated content enclosed within <TITLE>...</TITLE> and <PREVIEW>...</PREVIEW> tags.
2. No explanations, no markdown backticks outside the tags, no extra words.
3. Preserve URLs, links, numbers, product names and proper nouns where appropriate.
4. Preserve original line breaks and structure within <PREVIEW>.
5. If a section is already in the requested language, keep it unchanged within its tag.`

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
	Stream      bool          `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

var (
	titleTagPattern   = regexp.MustCompile(`(?is)<TITLE>(.*?)</TITLE>`)
	previewTagPattern = regexp.MustCompile(`(?is)<PREVIEW>(.*?)</PREVIEW>`)
)

func parseTaggedTranslation(content string) (title, preview string, ok bool) {
	titleMatch := titleTagPattern.FindStringSubmatch(content)
	previewMatch := previewTagPattern.FindStringSubmatch(content)

	if len(titleMatch) >= 2 && len(previewMatch) >= 2 {
		return strings.TrimSpace(titleMatch[1]), strings.TrimSpace(previewMatch[1]), true
	}
	return "", "", false
}

// Translate implements Translator for single string translation.
func (t *LLMTranslator) Translate(ctx context.Context, text, targetLang string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return text, nil
	}

	if IsSameLanguage(text, targetLang) {
		return text, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	body := chatRequest{
		Model: t.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf("Target language: %s\n\nText:\n%s", LanguageName(targetLang), text)},
		},
		Temperature: 0,
		MaxTokens:   maxTokensFor(text),
		Stream:      false,
	}

	return t.doRequest(ctx, body, targetLang)
}

// TranslateContent translates both title and preview in a single LLM request to save tokens and latency.
func (t *LLMTranslator) TranslateContent(ctx context.Context, title, preview, targetLang string) (string, string, error) {
	trimmedTitle := strings.TrimSpace(title)
	trimmedPreview := strings.TrimSpace(preview)

	titleNeedsTrans := trimmedTitle != "" && !IsSameLanguage(trimmedTitle, targetLang)
	previewNeedsTrans := trimmedPreview != "" && !IsSameLanguage(trimmedPreview, targetLang)

	if !titleNeedsTrans && !previewNeedsTrans {
		return title, preview, nil
	}

	if !titleNeedsTrans {
		transPreview, err := t.Translate(ctx, preview, targetLang)
		if err != nil {
			return title, preview, err
		}
		return title, transPreview, nil
	}

	if !previewNeedsTrans {
		transTitle, err := t.Translate(ctx, title, targetLang)
		if err != nil {
			return title, preview, err
		}
		return transTitle, preview, nil
	}

	// Both title and preview require translation -> Single combined request
	prompt := fmt.Sprintf(
		"Target language: %s\n\n<TITLE>\n%s\n</TITLE>\n\n<PREVIEW>\n%s\n</PREVIEW>",
		LanguageName(targetLang), trimmedTitle, trimmedPreview,
	)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	combinedText := trimmedTitle + "\n" + trimmedPreview
	body := chatRequest{
		Model: t.model,
		Messages: []chatMessage{
			{Role: "system", Content: combinedSystemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: 0,
		MaxTokens:   maxTokensFor(combinedText),
		Stream:      false,
	}

	respContent, err := t.doRequest(ctx, body, targetLang)
	if err != nil {
		return title, preview, err
	}

	parsedTitle, parsedPreview, ok := parseTaggedTranslation(respContent)
	if !ok {
		// Fallback: If model didn't enclose in tags, translate separately as fallback
		transTitle, err1 := t.Translate(ctx, title, targetLang)
		if err1 != nil {
			transTitle = title
		}
		transPreview, err2 := t.Translate(ctx, preview, targetLang)
		if err2 != nil {
			transPreview = preview
		}
		return transTitle, transPreview, nil
	}

	return parsedTitle, parsedPreview, nil
}

func (t *LLMTranslator) doRequest(ctx context.Context, body chatRequest, targetLang string) (string, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if t.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
	}
	if t.httpReferer != "" {
		req.Header.Set("HTTP-Referer", t.httpReferer)
	}
	if t.xTitle != "" {
		req.Header.Set("X-Title", t.xTitle)
	}

	resp, err := t.httpClient.Client().Do(req)
	if err != nil {
		return "", fmt.Errorf("translate request for target lang %s (%s) failed (%s): %w", targetLang, LanguageName(targetLang), t.baseURL+"/chat/completions", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		var apiErr chatResponse
		_ = json.Unmarshal(data, &apiErr)
		if apiErr.Error != nil {
			return "", fmt.Errorf(
				"translate api error (target lang %s [%s], status %d, url %s): %s",
				targetLang, LanguageName(targetLang), resp.StatusCode, t.baseURL+"/chat/completions", apiErr.Error.Message,
			)
		}
		return "", fmt.Errorf(
			"translate api error (target lang %s [%s], status %d, url %s): %s",
			targetLang, LanguageName(targetLang), resp.StatusCode, t.baseURL+"/chat/completions", truncate(string(data), 500),
		)
	}

	var parsed chatResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("translate api returned no choices for target lang %s (%s)", targetLang, LanguageName(targetLang))
	}
	out := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if out == "" {
		return "", fmt.Errorf("translate api returned empty translation for target lang %s (%s)", targetLang, LanguageName(targetLang))
	}
	return out, nil
}

// maxTokensFor budgets the output token limit from the input length. The
// translated text can be longer than the original, so a generous margin is
// used while staying well under common model limits.
func maxTokensFor(text string) int {
	n := len([]rune(text))*2 + 300
	if n < 512 {
		return 512
	}
	if n > 4096 {
		return 4096
	}
	return n
}

// truncate limits a string to n runes for log/error readability.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}
