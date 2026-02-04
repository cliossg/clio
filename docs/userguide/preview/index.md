# Preview

Preview generates your static site and serves it locally so you can see how it looks before publishing. Click the **Preview** card on the [site dashboard](../sites/dashboard/index.md) to generate and open the preview.

## How It Works

Clicking Preview does two things:

1. Generates the static site from your content, sections, layouts, and settings
2. Opens a new browser tab with the preview at `http://{slug}.localhost:3000/`

Each site gets its own subdomain based on its slug. If your site slug is `my-blog`, the preview is served at `http://my-blog.localhost:3000/`.

The preview server runs on port 3000 by default. This is separate from the Clio dashboard (port 8080).

## What Gets Generated

The preview builds the complete site using all published content. This includes:

- Index pages with content listings
- Individual content pages
- Section index pages
- Static assets (CSS, JavaScript)
- Images
- Contact forms (if enabled)
- Sitemaps and robots.txt

Draft content and content with a future publish date are excluded, just as they would be on the live site. See the [Content](../content/index.md) guide for details on publishing.

## Related Settings

These settings in the [Settings](../sites/dashboard/index.md#settings) page affect how the site is generated:

| Setting | Description | Default |
|---|---|---|
| **Site base URL** | The full base URL for the site (e.g. `https://example.com`) | `https://example.com` |
| **Site base path** | Base path for subpath hosting (e.g. `/blog` for GitHub Pages project sites) | `/` |
| **Site description** | Description shown in the hero area and HTML meta tags | |
| **Index max items** | Maximum number of items shown on index pages | `9` |
| **Blocks enabled** | Show related content blocks at the bottom of content pages | `true` |
| **Blocks max items** | Maximum items in a related content block | `5` |
| **Blocks multi-section** | Include related content from other sections | `true` |
| **Blocks background color** | Background color for related content blocks | `#f0f4f8` |

Other feature-specific settings (Google Analytics, cookie banner, search, forms) also affect the generated output. See their respective guides for details.
