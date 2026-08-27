package translate

import (
	"strings"
	"unicode"
)

// IsSameLanguage checks if the given text is already in the target language.
// When it returns true, translation can be safely skipped, saving LLM tokens.
func IsSameLanguage(text, targetLang string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return true
	}
	targetLang = strings.ToLower(strings.TrimSpace(targetLang))

	switch targetLang {
	case "zh", "zh-cn", "zh-hans", "zh-sg", "zh-tw", "zh-hant", "zh-hk":
		return isChinese(text)
	case "en", "en-us", "en-gb":
		return isEnglish(text)
	case "ja":
		return isJapanese(text)
	case "ko":
		return isKorean(text)
	case "ru":
		return isRussian(text)
	default:
		return false
	}
}

func isChinese(text string) bool {
	var hanCount, letterCount int
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			hanCount++
		} else if unicode.IsLetter(r) {
			letterCount++
		}
	}
	if hanCount > 0 && letterCount == 0 {
		return true
	}
	if hanCount >= 2 && float64(hanCount)/float64(hanCount+letterCount) >= 0.3 {
		return true
	}
	return false
}

func isEnglish(text string) bool {
	var asciiLetterCount, nonAsciiLetterCount int
	for _, r := range text {
		if r <= 127 {
			if unicode.IsLetter(r) {
				asciiLetterCount++
			}
		} else if unicode.IsLetter(r) {
			nonAsciiLetterCount++
		}
	}
	if asciiLetterCount > 0 && nonAsciiLetterCount == 0 {
		return true
	}
	return false
}

func isJapanese(text string) bool {
	var kanaCount int
	for _, r := range text {
		if unicode.Is(unicode.Hiragana, r) || unicode.Is(unicode.Katakana, r) {
			kanaCount++
		}
	}
	return kanaCount >= 2
}

func isKorean(text string) bool {
	var hangulCount int
	for _, r := range text {
		if unicode.Is(unicode.Hangul, r) {
			hangulCount++
		}
	}
	return hangulCount >= 2
}

func isRussian(text string) bool {
	var cyrillicCount int
	for _, r := range text {
		if unicode.Is(unicode.Cyrillic, r) {
			cyrillicCount++
		}
	}
	return cyrillicCount >= 2
}
