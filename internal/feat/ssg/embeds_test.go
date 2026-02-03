package ssg

import (
	"strings"
	"testing"
)

func TestEmbedConfigToHTML(t *testing.T) {
	tests := []struct {
		name    string
		config  EmbedConfig
		want    string
		wantErr bool
	}{
		{
			name: "youtube basic",
			config: EmbedConfig{
				Provider: "youtube",
				ID:       "dQw4w9WgXcQ",
			},
			want:    `<div class="embed-container ratio-16-9"><iframe src="https://www.youtube.com/embed/dQw4w9WgXcQ" title="YouTube video" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen loading="lazy"></iframe></div>`,
			wantErr: false,
		},
		{
			name: "youtube with custom ratio",
			config: EmbedConfig{
				Provider: "youtube",
				ID:       "abc123",
				Ratio:    "4:3",
			},
			want:    `<div class="embed-container ratio-4-3"><iframe src="https://www.youtube.com/embed/abc123" title="YouTube video" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen loading="lazy"></iframe></div>`,
			wantErr: false,
		},
		{
			name: "youtube with custom title",
			config: EmbedConfig{
				Provider: "youtube",
				ID:       "xyz789",
				Title:    "My Custom Video",
			},
			want:    `<div class="embed-container ratio-16-9"><iframe src="https://www.youtube.com/embed/xyz789" title="My Custom Video" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen loading="lazy"></iframe></div>`,
			wantErr: false,
		},
		{
			name: "vimeo basic",
			config: EmbedConfig{
				Provider: "vimeo",
				ID:       "123456789",
			},
			want:    `<div class="embed-container ratio-16-9"><iframe src="https://player.vimeo.com/video/123456789" title="Vimeo video" allow="autoplay; fullscreen; picture-in-picture" allowfullscreen loading="lazy"></iframe></div>`,
			wantErr: false,
		},
		{
			name: "tiktok with vertical ratio",
			config: EmbedConfig{
				Provider: "tiktok",
				ID:       "7234567890123456789",
				Ratio:    "9:16",
			},
			want:    `<div class="embed-container ratio-9-16"><iframe src="https://www.tiktok.com/embed/v2/7234567890123456789" title="TikTok video" allowfullscreen loading="lazy"></iframe></div>`,
			wantErr: false,
		},
		{
			name: "soundcloud with full url",
			config: EmbedConfig{
				Provider: "soundcloud",
				ID:       "https://soundcloud.com/artist/track-name",
			},
			want:    `w.soundcloud.com/player/?url=https%3A%2F%2Fsoundcloud.com%2Fartist%2Ftrack-name`,
			wantErr: false,
		},
		{
			name: "soundcloud with path only",
			config: EmbedConfig{
				Provider: "soundcloud",
				ID:       "artist/track-name",
			},
			want:    `w.soundcloud.com/player/?url=https%3A%2F%2Fsoundcloud.com%2Fartist%2Ftrack-name`,
			wantErr: false,
		},
		{
			name: "missing provider",
			config: EmbedConfig{
				ID: "abc123",
			},
			wantErr: true,
		},
		{
			name: "missing id",
			config: EmbedConfig{
				Provider: "youtube",
			},
			wantErr: true,
		},
		{
			name: "unsupported provider",
			config: EmbedConfig{
				Provider: "dailymotion",
				ID:       "abc123",
			},
			wantErr: true,
		},
		{
			name: "invalid ratio fallback",
			config: EmbedConfig{
				Provider: "youtube",
				ID:       "abc123",
				Ratio:    "invalid",
			},
			want:    `ratio-16-9`,
			wantErr: false,
		},
		{
			name: "case insensitive provider",
			config: EmbedConfig{
				Provider: "YouTube",
				ID:       "abc123",
			},
			want:    `youtube.com/embed/abc123`,
			wantErr: false,
		},
		{
			name: "html provider with code",
			config: EmbedConfig{
				Provider: "html",
				Code:     `<blockquote class="twitter-tweet"><p>Hello</p></blockquote>`,
			},
			want:    `<div class="embed-html"><blockquote class="twitter-tweet"><p>Hello</p></blockquote></div>`,
			wantErr: false,
		},
		{
			name: "html provider without code",
			config: EmbedConfig{
				Provider: "html",
			},
			wantErr: true,
		},
		{
			name: "html provider with empty code",
			config: EmbedConfig{
				Provider: "html",
				Code:     "   ",
			},
			wantErr: true,
		},
		{
			name: "html provider does not require id",
			config: EmbedConfig{
				Provider: "html",
				Code:     "<div>test</div>",
			},
			want:    `<div class="embed-html"><div>test</div></div>`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.config.ToHTML()
			if (err != nil) != tt.wantErr {
				t.Errorf("ToHTML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.Contains(got, tt.want) {
				t.Errorf("ToHTML() = %v, want substring %v", got, tt.want)
			}
		})
	}
}

func TestProcessEmbeds(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "youtube embed block",
			html: `<p>Hello</p><pre><code class="language-embed">provider: youtube
id: dQw4w9WgXcQ</code></pre><p>World</p>`,
			want: `<div class="embed-container ratio-16-9"><iframe src="https://www.youtube.com/embed/dQw4w9WgXcQ"`,
		},
		{
			name: "vimeo embed with ratio",
			html: `<pre><code class="language-embed">provider: vimeo
id: 123456
ratio: 4:3</code></pre>`,
			want: `<div class="embed-container ratio-4-3"><iframe src="https://player.vimeo.com/video/123456"`,
		},
		{
			name: "tiktok embed",
			html: `<pre><code class="language-embed">provider: tiktok
id: 7234567890
ratio: 9:16</code></pre>`,
			want: `<div class="embed-container ratio-9-16"><iframe src="https://www.tiktok.com/embed/v2/7234567890"`,
		},
		{
			name: "no embed block unchanged",
			html: `<p>Hello</p><pre><code class="language-go">func main() {}</code></pre>`,
			want: `<pre><code class="language-go">func main() {}</code></pre>`,
		},
		{
			name: "invalid provider unchanged",
			html: `<pre><code class="language-embed">provider: unknown
id: abc123</code></pre>`,
			want: `<pre><code class="language-embed">`,
		},
		{
			name: "missing id unchanged",
			html: `<pre><code class="language-embed">provider: youtube</code></pre>`,
			want: `<pre><code class="language-embed">`,
		},
		{
			name: "html entities decoded",
			html: `<pre><code class="language-embed">provider: youtube
id: abc&amp;def</code></pre>`,
			want: `youtube.com/embed/abc&def`,
		},
		{
			name: "multiple embeds",
			html: `<pre><code class="language-embed">provider: youtube
id: vid1</code></pre><pre><code class="language-embed">provider: vimeo
id: vid2</code></pre>`,
			want: `youtube.com/embed/vid1`,
		},
		{
			name: "html embed with separator",
			html: `<pre><code class="language-embed">provider: html
---
&lt;blockquote class=&quot;twitter-tweet&quot;&gt;&lt;p&gt;Hello&lt;/p&gt;&lt;/blockquote&gt;</code></pre>`,
			want: `<div class="embed-html"><blockquote class="twitter-tweet"><p>Hello</p></blockquote></div>`,
		},
		{
			name: "html embed with script",
			html: `<pre><code class="language-embed">provider: html
---
&lt;blockquote&gt;tweet&lt;/blockquote&gt;
&lt;script async src=&quot;https://platform.twitter.com/widgets.js&quot;&gt;&lt;/script&gt;</code></pre>`,
			want: `<div class="embed-html"><blockquote>tweet</blockquote>`,
		},
		{
			name: "html embed without code unchanged",
			html: `<pre><code class="language-embed">provider: html</code></pre>`,
			want: `<pre><code class="language-embed">`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processEmbeds(tt.html)
			if !strings.Contains(got, tt.want) {
				t.Errorf("processEmbeds() = %v, want substring %v", got, tt.want)
			}
		})
	}
}

func TestUnescapeHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"amp", "foo&amp;bar", "foo&bar"},
		{"lt", "foo&lt;bar", "foo<bar"},
		{"gt", "foo&gt;bar", "foo>bar"},
		{"quot", "foo&quot;bar", "foo\"bar"},
		{"apos", "foo&#39;bar", "foo'bar"},
		{"multiple", "&lt;div&gt;&amp;&quot;test&#39;", "<div>&\"test'"},
		{"none", "plain text", "plain text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unescapeHTML(tt.input)
			if got != tt.want {
				t.Errorf("unescapeHTML(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidRatios(t *testing.T) {
	expected := map[string]string{
		"16:9": "ratio-16-9",
		"4:3":  "ratio-4-3",
		"1:1":  "ratio-1-1",
		"9:16": "ratio-9-16",
	}

	for ratio, class := range expected {
		if validRatios[ratio] != class {
			t.Errorf("validRatios[%q] = %q, want %q", ratio, validRatios[ratio], class)
		}
	}
}

func TestEmbedProviders(t *testing.T) {
	requiredProviders := []string{"youtube", "vimeo", "tiktok", "soundcloud"}

	for _, p := range requiredProviders {
		provider, ok := EmbedProviders[p]
		if !ok {
			t.Errorf("missing provider: %s", p)
			continue
		}
		if provider.Name == "" {
			t.Errorf("provider %s has empty name", p)
		}
		if provider.URLPattern == "" {
			t.Errorf("provider %s has empty URL pattern", p)
		}
	}
}
