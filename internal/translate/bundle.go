package translate

import (
	"embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// Bundle holds all translation data and implements Localizer.
type Bundle struct {
	fs    embed.FS
	langs map[string]*languageData // lang → data (preloaded)
}

// NewBundle creates and preloads all languages from the embedded FS.
func NewBundle(fs embed.FS) *Bundle {
	b := &Bundle{
		fs:    fs,
		langs: make(map[string]*languageData),
	}

	b.loadAll()
	return b
}

// loadAll preloads every language found in share/translations/
func (b *Bundle) loadAll() {
	entries, err := b.fs.ReadDir("share/translations")
	if err != nil {
		logrus.WithError(err).Fatal("Failed to read translations directory")
	}

	fullCount := 0
	partialCount := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		lang := entry.Name()
		if lang == "messages" || lang == "." || lang == ".." {
			continue
		}

		data := b.loadLanguage(lang)
		b.langs[lang] = data

		isFull := false
		if meta, err := b.fs.ReadFile(fmt.Sprintf("share/translations/%s/metadata.json", lang)); err == nil {
			var m struct {
				FullTranslation bool `json:"full_translation"`
			}
			if json.Unmarshal(meta, &m) == nil {
				isFull = m.FullTranslation
			}
		}

		if isFull {
			fullCount++
		} else {
			partialCount++
		}

		logrus.WithFields(logrus.Fields{
			"lang":  lang,
			"full":  isFull,
			"files": len(data.messages) + len(data.conditions),
		}).Debugf("Loaded language %s", lang)
	}

	total := len(b.langs)
	logrus.WithFields(logrus.Fields{
		"total":   total,
		"full":    fullCount,
		"partial": partialCount,
	}).Infof("Translations loaded: %d languages (%d full, %d partial)", total, fullCount, partialCount)
}

// Text implements Localizer.
func (b *Bundle) Text(lang, key string) string {
	lang = normalizeLang(lang)

	data, ok := b.langs[lang]
	if !ok {
		// fallback to English
		if lang != "en" {
			return b.Text("en", key)
		}
		return key
	}

	// Direct message / view key
	if s, ok := data.messages[key]; ok && s != "" {
		return s
	}

	// Condition by code
	if code, err := strconv.Atoi(strings.TrimSpace(key)); err == nil {
		if s, ok := data.conditions[code]; ok && s != "" {
			return s
		}
	}

	// Condition by English name
	lower := strings.ToLower(strings.TrimSpace(key))
	if s, ok := data.byEnglish[lower]; ok && s != "" {
		return s
	}

	// Final fallback to English
	if lang != "en" {
		return b.Text("en", key)
	}
	return key
}

// File implements Localizer.
func (b *Bundle) File(lang, name string) (string, error) {
	lang = normalizeLang(lang)
	p := fmt.Sprintf("share/translations/%s/%s", lang, name)

	data, err := b.fs.ReadFile(p)
	if err != nil {
		if lang != "en" {
			return b.File("en", name)
		}
		return "", fmt.Errorf("file %s not found for language %s", name, lang)
	}
	return string(data), nil
}

// loadLanguage loads a single language (used during initialization)
func (b *Bundle) loadLanguage(lang string) *languageData {
	ld := &languageData{
		messages:   make(map[string]string),
		conditions: make(map[int]string),
		byEnglish:  make(map[string]string),
	}

	base := fmt.Sprintf("share/translations/%s/", lang)

	// Load messages, v1, v2
	for _, filename := range []string{"messages.json", "v1.json", "v2.json"} {
		data, err := b.fs.ReadFile(base + filename)
		if err == nil {
			var m map[string]string
			if json.Unmarshal(data, &m) == nil {
				for k, v := range m {
					ld.messages[k] = v
				}
			}
		}
	}

	// Load conditions.txt
	if data, err := b.fs.ReadFile(base + "conditions.txt"); err == nil {
		parseConditions(data, ld.conditions, ld.byEnglish)
	}

	return ld
}

func normalizeLang(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		return "en"
	}
	return lang
}
