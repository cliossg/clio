# Embeds

Embeds let you include videos and audio from external platforms like YouTube, Vimeo, TikTok, and SoundCloud directly in your content using a simple Markdown syntax.

## Why use Embeds?

- **Rich media**: Add videos and audio without hosting files yourself
- **Responsive**: Embeds automatically adapt to screen size
- **Simple syntax**: No HTML knowledge required
- **Preview support**: See placeholders while editing

## Quick Start

Add a YouTube video to your content:

````markdown
```embed
provider: youtube
id: dQw4w9WgXcQ
```
````

When your site is generated, this becomes a responsive embedded video player.

## Syntax

Embeds use a fenced code block with the `embed` language:

````markdown
```embed
provider: youtube
id: VIDEO_ID
ratio: 16:9
title: My Video Title
```
````

### Required fields

| Field | Description |
|-------|-------------|
| `provider` | The platform: `youtube`, `vimeo`, `tiktok`, or `soundcloud` |
| `id` | The video or track ID from the platform |

### Optional fields

| Field | Description | Default |
|-------|-------------|---------|
| `ratio` | Aspect ratio | `16:9` |
| `title` | Accessibility title | Platform name + "video" |

## Supported Providers

### YouTube

```embed
provider: youtube
id: dQw4w9WgXcQ
```

The ID is the part after `v=` in a YouTube URL:
- `https://www.youtube.com/watch?v=dQw4w9WgXcQ` → ID is `dQw4w9WgXcQ`

### Vimeo

```embed
provider: vimeo
id: 123456789
```

The ID is the number in a Vimeo URL:
- `https://vimeo.com/123456789` → ID is `123456789`

### TikTok

```embed
provider: tiktok
id: 7234567890123456789
ratio: 9:16
```

The ID is the long number in a TikTok URL:
- `https://www.tiktok.com/@user/video/7234567890123456789` → ID is `7234567890123456789`

> **Tip**: Use `ratio: 9:16` for TikTok videos since they're vertical.

### SoundCloud

```embed
provider: soundcloud
id: https://soundcloud.com/artist/track-name
```

For SoundCloud, paste the full URL or just the path:
- Full URL: `https://soundcloud.com/artist/track-name`
- Path only: `artist/track-name`
- Playlists: `artist/sets/playlist-name`

## Aspect Ratios

Choose the ratio that matches your content:

| Ratio | Use case | Padding |
|-------|----------|---------|
| `16:9` | Standard video (YouTube, Vimeo) | 56.25% |
| `4:3` | Older video formats | 75% |
| `1:1` | Square video | 100% |
| `9:16` | Vertical video (TikTok, Reels) | 177.78% |

The default is `16:9`, which works for most videos.

### Vertical video example

````markdown
```embed
provider: tiktok
id: 7234567890123456789
ratio: 9:16
```
````

Vertical embeds are constrained to a maximum width of 400px to prevent them from taking over the page.

## Editor Preview

While editing in Clio, embeds appear as placeholders showing the provider and ID:

```
┌─────────────────────────────┐
│            ▶               │
│     youtube: dQw4w9WgXcQ   │
└─────────────────────────────┘
```

The actual video player only appears in the generated site. This keeps the editor fast and avoids loading external resources while you write.

## Common Workflows

### Adding a video to an article

1. Find the video on YouTube, Vimeo, etc.
2. Copy the video ID from the URL
3. Add the embed block to your Markdown:

````markdown
Here's a great tutorial:

```embed
provider: youtube
id: abc123xyz
```

As you can see in the video...
````

### Mixing content types

You can include multiple embeds in one article:

````markdown
## Video Tutorial

```embed
provider: youtube
id: tutorial123
```

## The Soundtrack

```embed
provider: soundcloud
id: tracks/456789
```
````

### Adding accessibility titles

For better accessibility, add a descriptive title:

````markdown
```embed
provider: youtube
id: abc123
title: Introduction to Clio - Getting Started Tutorial
```
````

This title appears in the iframe's `title` attribute for screen readers.

## Generated HTML

When your site is generated, the embed block becomes:

```html
<div class="embed-container ratio-16-9">
  <iframe
    src="https://www.youtube.com/embed/dQw4w9WgXcQ"
    title="YouTube video"
    allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
    allowfullscreen
    loading="lazy">
  </iframe>
</div>
```

The embed uses:
- Responsive container with padding-based aspect ratio
- Lazy loading for better performance
- Appropriate `allow` attributes per provider
- Accessible title attribute

## Troubleshooting

### Embed doesn't render

Check that:
- The code block uses exactly `embed` as the language
- `provider` and `id` fields are present
- The provider name is lowercase

### Video doesn't play

The embed creates a standard iframe. If the video doesn't play:
- Verify the video ID is correct
- Check if the video is available in your region
- Ensure the video allows embedding (some videos disable this)

### Wrong aspect ratio

If the video appears stretched or has black bars:
- YouTube/Vimeo: usually `16:9`
- TikTok/Reels: use `9:16`
- Square content: use `1:1`
- Older content: try `4:3`

### Preview shows placeholder but generated site is empty

If the preview shows the placeholder correctly but the generated site doesn't render the embed, check the Markdown syntax. The fenced code block needs exactly three backticks and the word `embed`:

````markdown
```embed
provider: youtube
id: abc123
```
````
