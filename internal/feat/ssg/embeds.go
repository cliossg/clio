package ssg

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type EmbedConfig struct {
	Provider string `yaml:"provider"`
	ID       string `yaml:"id"`
	Ratio    string `yaml:"ratio"`
	Title    string `yaml:"title"`
}

type EmbedProvider struct {
	Name       string
	URLPattern string
	AllowAttr  string
}

var EmbedProviders = map[string]EmbedProvider{
	"youtube": {
		Name:       "YouTube",
		URLPattern: "https://www.youtube.com/embed/%s",
		AllowAttr:  "accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture",
	},
	"vimeo": {
		Name:       "Vimeo",
		URLPattern: "https://player.vimeo.com/video/%s",
		AllowAttr:  "autoplay; fullscreen; picture-in-picture",
	},
	"tiktok": {
		Name:       "TikTok",
		URLPattern: "https://www.tiktok.com/embed/v2/%s",
		AllowAttr:  "",
	},
	"soundcloud": {
		Name:       "SoundCloud",
		URLPattern: "https://w.soundcloud.com/player/?url=%s",
		AllowAttr:  "",
	},
}

var validRatios = map[string]string{
	"16:9": "ratio-16-9",
	"4:3":  "ratio-4-3",
	"1:1":  "ratio-1-1",
	"9:16": "ratio-9-16",
}

func (e *EmbedConfig) ToHTML() (string, error) {
	if e.Provider == "" {
		return "", fmt.Errorf("provider is required")
	}
	if e.ID == "" {
		return "", fmt.Errorf("id is required")
	}

	provider, ok := EmbedProviders[strings.ToLower(e.Provider)]
	if !ok {
		return "", fmt.Errorf("unsupported provider: %s", e.Provider)
	}

	ratio := e.Ratio
	if ratio == "" {
		ratio = "16:9"
	}

	ratioClass, ok := validRatios[ratio]
	if !ok {
		ratioClass = "ratio-16-9"
	}

	id := e.ID
	if strings.ToLower(e.Provider) == "soundcloud" {
		if !strings.HasPrefix(id, "http") {
			id = "https://soundcloud.com/" + id
		}
		id = url.QueryEscape(id)
	}

	embedURL := fmt.Sprintf(provider.URLPattern, id)

	title := e.Title
	if title == "" {
		title = fmt.Sprintf("%s video", provider.Name)
	}

	var allowAttr string
	if provider.AllowAttr != "" {
		allowAttr = fmt.Sprintf(` allow="%s"`, provider.AllowAttr)
	}

	return fmt.Sprintf(
		`<div class="embed-container %s"><iframe src="%s" title="%s"%s allowfullscreen loading="lazy"></iframe></div>`,
		ratioClass, embedURL, title, allowAttr,
	), nil
}

var embedCodeBlockRegex = regexp.MustCompile(`<pre><code class="language-embed">([\s\S]*?)</code></pre>`)

func processEmbeds(html string) string {
	return embedCodeBlockRegex.ReplaceAllStringFunc(html, func(match string) string {
		submatches := embedCodeBlockRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		yamlContent := strings.TrimSpace(submatches[1])
		yamlContent = unescapeHTML(yamlContent)

		var config EmbedConfig
		if err := yaml.Unmarshal([]byte(yamlContent), &config); err != nil {
			return match
		}

		embedHTML, err := config.ToHTML()
		if err != nil {
			return match
		}

		return embedHTML
	})
}

func unescapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	return s
}
