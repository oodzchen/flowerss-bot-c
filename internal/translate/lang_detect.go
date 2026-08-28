package translate

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	htmlTagRegex = regexp.MustCompile(`<[^>]+>`)
	urlRegex     = regexp.MustCompile(`https?://[^\s]+`)
	xmlLangRegex = regexp.MustCompile(`(?i)(?:xml:)?lang=["']([a-zA-Z0-9_-]+)["']`)
)

// NormalizeLangTag normalizes a language tag (e.g. "zh_CN" -> "zh-cn", "EN-US" -> "en-us").
func NormalizeLangTag(tag string) string {
	tag = strings.TrimSpace(tag)
	tag = strings.Trim(tag, `"'`)
	tag = strings.ToLower(tag)
	tag = strings.ReplaceAll(tag, "_", "-")
	return tag
}

// ExtractXMLLanguage extracts a declared language attribute (e.g. lang="zh-CN" or xml:lang="en")
// from raw XML/HTML text.
func ExtractXMLLanguage(s string) string {
	if s == "" {
		return ""
	}
	m := xmlLangRegex.FindStringSubmatch(s)
	if len(m) >= 2 {
		return NormalizeLangTag(m[1])
	}
	return ""
}

// MatchLanguage reports whether sourceLang matches targetLang according to BCP 47/ISO language codes.
func MatchLanguage(srcLang, targetLang string) bool {
	src := NormalizeLangTag(srcLang)
	tgt := NormalizeLangTag(targetLang)
	if src == "" || tgt == "" {
		return false
	}
	if src == tgt {
		return true
	}

	srcParts := strings.Split(src, "-")
	tgtParts := strings.Split(tgt, "-")
	srcBase := srcParts[0]
	tgtBase := tgtParts[0]

	// Handle Chinese variants
	if tgtBase == "zh" || srcBase == "zh" {
		if tgt == "zh" || src == "zh" || tgt == "cmn" || src == "cmn" || tgt == "chinese" || src == "chinese" {
			return (srcBase == "zh" || src == "cmn" || src == "chinese") && (tgtBase == "zh" || tgt == "cmn" || tgt == "chinese")
		}
		// Simplified Chinese groups
		isSrcSimp := src == "zh-cn" || src == "zh-hans" || src == "zh-sg"
		isTgtSimp := tgt == "zh-cn" || tgt == "zh-hans" || tgt == "zh-sg"
		if isSrcSimp && isTgtSimp {
			return true
		}
		// Traditional Chinese groups
		isSrcTrad := src == "zh-tw" || src == "zh-hant" || src == "zh-hk" || src == "zh-mo"
		isTgtTrad := tgt == "zh-tw" || tgt == "zh-hant" || tgt == "zh-hk" || tgt == "zh-mo"
		if isSrcTrad && isTgtTrad {
			return true
		}
		// Generic "zh" target accepts any Chinese
		if tgt == "zh" && (isSrcSimp || isSrcTrad) {
			return true
		}
		return false
	}

	// Base language code equality (e.g. en-US matches en, ja-JP matches ja, ru-RU matches ru)
	return srcBase == tgtBase
}

type scriptStats struct {
	hanCount        int
	hiraganaCount   int
	katakanaCount   int
	hangulCount     int
	cyrillicCount   int
	arabicCount     int
	hebrewCount     int
	greekCount      int
	thaiCount       int
	devanagariCount int
	latinAsciiCount int
	latinExtCount   int
	cjkPunctCount   int
	totalLetters    int
}

func cleanTextForDetection(s string) string {
	s = htmlTagRegex.ReplaceAllString(s, " ")
	s = urlRegex.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func isCJKPunct(r rune) bool {
	switch r {
	case '，', '。', '！', '？', '【', '】', '《', '》', '“', '”', '‘', '’', '、', '；', '：', '…', '—', '〜', '（', '）', '「', '」', '『', '』', '・':
		return true
	default:
		return false
	}
}

func collectScriptStats(text string) scriptStats {
	var s scriptStats
	for _, r := range text {
		switch {
		case unicode.Is(unicode.Han, r):
			s.hanCount++
			s.totalLetters++
		case unicode.Is(unicode.Hiragana, r):
			s.hiraganaCount++
			s.totalLetters++
		case unicode.Is(unicode.Katakana, r):
			s.katakanaCount++
			s.totalLetters++
		case unicode.Is(unicode.Hangul, r):
			s.hangulCount++
			s.totalLetters++
		case unicode.Is(unicode.Cyrillic, r):
			s.cyrillicCount++
			s.totalLetters++
		case unicode.Is(unicode.Arabic, r):
			s.arabicCount++
			s.totalLetters++
		case unicode.Is(unicode.Hebrew, r):
			s.hebrewCount++
			s.totalLetters++
		case unicode.Is(unicode.Greek, r):
			s.greekCount++
			s.totalLetters++
		case unicode.Is(unicode.Thai, r):
			s.thaiCount++
			s.totalLetters++
		case unicode.Is(unicode.Devanagari, r):
			s.devanagariCount++
			s.totalLetters++
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			s.latinAsciiCount++
			s.totalLetters++
		case unicode.Is(unicode.Latin, r) && r > 127:
			s.latinExtCount++
			s.totalLetters++
		default:
			if isCJKPunct(r) {
				s.cjkPunctCount++
			} else if unicode.IsLetter(r) {
				s.totalLetters++
			}
		}
	}
	return s
}

func detectScriptLanguage(text string, stats scriptStats) string {
	if stats.totalLetters == 0 {
		return ""
	}

	// 1. Japanese (Hiragana or Katakana present)
	kanaCount := stats.hiraganaCount + stats.katakanaCount
	if kanaCount >= 2 || (kanaCount >= 1 && stats.hanCount >= 1) {
		return "ja"
	}

	// 2. Korean (Hangul present)
	if stats.hangulCount >= 2 || (stats.hangulCount >= 1 && stats.totalLetters <= 6) {
		return "ko"
	}

	// 3. Chinese (Han characters present without Kana or Hangul)
	if stats.hiraganaCount == 0 && stats.katakanaCount == 0 && stats.hangulCount == 0 && stats.hanCount > 0 {
		nonHanNonLatin := stats.cyrillicCount + stats.arabicCount + stats.hebrewCount + stats.greekCount + stats.thaiCount + stats.devanagariCount
		if nonHanNonLatin == 0 {
			latinCount := stats.latinAsciiCount + stats.latinExtCount
			if latinCount == 0 {
				return "zh"
			}
			if stats.hanCount >= 2 && float64(stats.hanCount)/float64(stats.totalLetters) >= 0.20 {
				return "zh"
			}
			if stats.cjkPunctCount > 0 && stats.hanCount >= 1 {
				return "zh"
			}
		}
	}

	// 4. Cyrillic (Russian, Ukrainian, Bulgarian, etc.)
	if stats.cyrillicCount >= 2 || (stats.cyrillicCount >= 1 && stats.totalLetters <= 6) {
		return "ru"
	}

	// 5. Arabic script
	if stats.arabicCount >= 2 || (stats.arabicCount >= 1 && stats.totalLetters <= 6) {
		return "ar"
	}

	// 6. Hebrew script
	if stats.hebrewCount >= 2 || (stats.hebrewCount >= 1 && stats.totalLetters <= 6) {
		return "he"
	}

	// 7. Greek script
	if stats.greekCount >= 2 || (stats.greekCount >= 1 && stats.totalLetters <= 6) {
		return "el"
	}

	// 8. Thai script
	if stats.thaiCount >= 2 || (stats.thaiCount >= 1 && stats.totalLetters <= 6) {
		return "th"
	}

	// 9. Devanagari (Hindi, Marathi, etc.)
	if stats.devanagariCount >= 2 || (stats.devanagariCount >= 1 && stats.totalLetters <= 6) {
		return "hi"
	}

	// 10. Specific Latin language checks
	if isVietnamese(text) {
		return "vi"
	}
	if isGerman(text) {
		return "de"
	}
	if isSpanish(text) {
		return "es"
	}
	if isFrench(text) {
		return "fr"
	}

	// 11. Latin script (English or general Latin)
	totalLatin := stats.latinAsciiCount + stats.latinExtCount
	nonLatin := stats.hanCount + stats.hiraganaCount + stats.katakanaCount + stats.hangulCount + stats.cyrillicCount + stats.arabicCount + stats.hebrewCount + stats.greekCount + stats.thaiCount + stats.devanagariCount
	if totalLatin > 0 && nonLatin == 0 {
		if stats.latinExtCount == 0 || float64(stats.latinExtCount)/float64(totalLatin) <= 0.10 {
			return "en"
		}
		return "latin"
	}

	return ""
}

// IsSameLanguage checks if the given text is already in the target language.
// When it returns true, translation can be safely skipped, saving LLM tokens.
func IsSameLanguage(text, targetLang string) bool {
	return IsSameLanguageWithMeta(text, targetLang, "")
}

// IsSameLanguageWithMeta checks if the text (with optional XML metadata language hint)
// is already in targetLang.
func IsSameLanguageWithMeta(text, targetLang, metaLang string) bool {
	targetLang = NormalizeLangTag(targetLang)
	if targetLang == "" {
		return true // Translation disabled -> no translation needed
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return true
	}

	if metaLang == "" {
		metaLang = ExtractXMLLanguage(text)
	}
	metaLang = NormalizeLangTag(metaLang)

	cleaned := cleanTextForDetection(text)
	if cleaned == "" {
		return true
	}

	stats := collectScriptStats(cleaned)
	if stats.totalLetters == 0 {
		// Only numbers, timestamps, punctuation, or symbols -> No translation needed
		return true
	}

	detected := detectScriptLanguage(cleaned, stats)

	// 1. If metaLang is provided and matches targetLang:
	// Trust XML metadata (e.g. feed/item declared lang="zh-CN" and target is "zh")
	if metaLang != "" && MatchLanguage(metaLang, targetLang) {
		if detected != "" && isNonLatinScript(detected) && !MatchLanguage(detected, targetLang) {
			return false
		}
		return true
	}

	// 2. If metaLang is provided and explicitly belongs to a DIFFERENT language family:
	// (e.g. metaLang is "fr" and target is "en", or metaLang is "en" and target is "zh")
	if metaLang != "" && !MatchLanguage(metaLang, targetLang) {
		metaBase := strings.Split(metaLang, "-")[0]
		targetBase := strings.Split(targetLang, "-")[0]
		if metaBase != targetBase {
			return false
		}
	}

	// 3. Direct match between detected script language and target language
	if detected != "" && MatchLanguage(detected, targetLang) {
		return true
	}

	// 4. Script mismatch: If the text is clearly in a different script from target
	if detected != "" && detected != "latin" {
		targetBase := strings.Split(targetLang, "-")[0]
		if isNonLatinTarget(targetBase) && isNonLatinScript(detected) && detected != targetBase {
			return false
		}
		if detected == "en" && isNonLatinTarget(targetBase) {
			return false
		}
		if (detected == "fr" || detected == "de" || detected == "es" || detected == "vi") && targetBase == "en" {
			return false
		}
	}

	// 5. Target-specific checks
	targetBase := strings.Split(targetLang, "-")[0]
	switch targetBase {
	case "zh":
		return isChinese(cleaned, stats)
	case "en":
		return isEnglish(cleaned, stats)
	case "ja":
		return isJapanese(stats)
	case "ko":
		return isKorean(stats)
	case "ru":
		return isRussian(stats)
	case "de":
		return isGerman(cleaned)
	case "fr":
		return isFrench(cleaned)
	case "es":
		return isSpanish(cleaned)
	case "pt":
		return isPortuguese(cleaned)
	case "vi":
		return isVietnamese(cleaned)
	default:
		if metaLang != "" && MatchLanguage(metaLang, targetLang) {
			return true
		}
		return false
	}
}

func isNonLatinScript(script string) bool {
	switch script {
	case "zh", "ja", "ko", "ru", "ar", "he", "el", "th", "hi":
		return true
	default:
		return false
	}
}

func isNonLatinTarget(targetBase string) bool {
	switch targetBase {
	case "zh", "ja", "ko", "ru", "uk", "bg", "be", "ar", "fa", "ur", "he", "iw", "el", "th", "hi", "mr", "ne":
		return true
	default:
		return false
	}
}

func isChinese(text string, stats scriptStats) bool {
	if stats.hiraganaCount > 0 || stats.katakanaCount > 0 || stats.hangulCount > 0 {
		return false
	}
	if stats.hanCount == 0 {
		return false
	}
	nonHanNonLatin := stats.cyrillicCount + stats.arabicCount + stats.hebrewCount + stats.greekCount + stats.thaiCount + stats.devanagariCount
	if nonHanNonLatin > 0 {
		return false
	}
	latinCount := stats.latinAsciiCount + stats.latinExtCount
	if latinCount == 0 {
		return true
	}
	if stats.hanCount >= 2 && float64(stats.hanCount)/float64(stats.totalLetters) >= 0.20 {
		return true
	}
	if stats.cjkPunctCount > 0 && stats.hanCount >= 1 {
		return true
	}
	return false
}

func isEnglish(text string, stats scriptStats) bool {
	nonLatin := stats.hanCount + stats.hiraganaCount + stats.katakanaCount + stats.hangulCount + stats.cyrillicCount + stats.arabicCount + stats.hebrewCount + stats.greekCount + stats.thaiCount + stats.devanagariCount
	if nonLatin > 0 {
		return false
	}
	totalLatin := stats.latinAsciiCount + stats.latinExtCount
	if totalLatin == 0 {
		return false
	}
	if stats.latinExtCount == 0 {
		return true
	}
	if isFrench(text) || isGerman(text) || isSpanish(text) || isVietnamese(text) {
		return false
	}
	return float64(stats.latinExtCount)/float64(totalLatin) <= 0.10
}

func isJapanese(stats scriptStats) bool {
	kanaCount := stats.hiraganaCount + stats.katakanaCount
	return kanaCount >= 2 || (kanaCount >= 1 && stats.hanCount >= 1)
}

func isKorean(stats scriptStats) bool {
	return stats.hangulCount >= 2 || (stats.hangulCount >= 1 && stats.totalLetters <= 6)
}

func isRussian(stats scriptStats) bool {
	return stats.cyrillicCount >= 2 || (stats.cyrillicCount >= 1 && stats.totalLetters <= 6)
}

func isGerman(text string) bool {
	var deCount int
	for _, r := range text {
		switch r {
		case 'ä', 'ö', 'ü', 'ß', 'Ä', 'Ö', 'Ü':
			deCount++
		}
	}
	return deCount >= 1
}

func isFrench(text string) bool {
	var frDistinctCount int
	for _, r := range text {
		switch r {
		case 'è', 'ê', 'ë', 'à', 'â', 'ç', 'î', 'ï', 'ô', 'ù', 'û', 'œ', 'æ', 'È', 'Ê', 'Ë', 'À', 'Â', 'Ç', 'Î', 'Ï', 'Ô', 'Ù', 'Û', 'Œ', 'Æ':
			frDistinctCount++
		}
	}
	return frDistinctCount >= 1
}

func isSpanish(text string) bool {
	var esCount int
	for _, r := range text {
		switch r {
		case 'ñ', 'Ñ', '¿', '¡':
			esCount++
		}
	}
	return esCount >= 1
}

func isPortuguese(text string) bool {
	var ptCount int
	for _, r := range text {
		switch r {
		case 'ã', 'õ', 'Ã', 'Õ':
			ptCount++
		}
	}
	return ptCount >= 1
}

func isVietnamese(text string) bool {
	var viCount int
	for _, r := range text {
		switch r {
		case 'đ', 'Đ', 'ơ', 'ư', 'Ơ', 'Ư', 'ă', 'Ă', 'â', 'Â', 'ê', 'Ê', 'ô', 'Ô':
			viCount++
		}
	}
	return viCount >= 1
}
