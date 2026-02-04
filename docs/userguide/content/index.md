# Content

Content is where you manage all posts, pages, and articles for a site. Click the **Content** card on the [site dashboard](../sites/dashboard/index.md) to open the content list.

## The Content List

The content list shows all content items for the current site in a searchable, paginated table. Each row displays:

| Column | Description |
|---|---|
| **Title** | The content title (clickable) |
| **Section** | The section this content belongs to, or "None" if unassigned |
| **Kind** | The content type: page, article, or post |
| **Status** | Published (green) or Draft (yellow) |
| **Actions** | Edit and Delete buttons |

### Searching

The search box at the top filters the list dynamically as you type. Results update without reloading the page. The list is paginated, so if you have many content items, use search to narrow down results or navigate between pages.

---

## Creating Content

Click **New Content** in the top-right corner. The form has the following sections:

### Title

The display title for your content. This is also used to generate the URL slug.

### Header Image

An optional hero image displayed at the top of your content. The header image can only be uploaded after saving the content for the first time.

The **Light/Dark** toggle tells Clio about the brightness of the image. The default templates use this to choose a text color with good contrast for the frosted overlay that sits on top of the hero. If your image is mostly light, select **Light** so the overlay uses dark text. If the image is dark, select **Dark** for light text.

The image also includes fields for alt text and tags, visible when expanded.

### Content Editor

The main writing area uses Markdown with a split-pane view: your Markdown source on the left, a live preview on the right.

The editor toolbar provides formatting shortcuts:

| Button | What it does |
|---|---|
| **B** | Bold text |
| **I** | Italic text |
| **</>** | Inline code |
| **Link** | Insert a link |
| **Img** | Insert an image reference |
| **Embed** | Insert an embed block (YouTube, Vimeo, TikTok, SoundCloud). Available after first save. |
| **Form** | Insert a contact form block. Available after first save. |
| **Proofread** | Run AI-powered proofreading. See the [Proofread](../proofread/index.md) guide. |
| **Meta** | Toggle SEO metadata fields |
| **Zen** | Distraction-free writing mode. Hides the preview pane and shows only the Markdown editor. Includes a toggle for dark mode. |

### Content Images

Below the editor, the **Content Images** section lets you upload images and insert them into your content. Click an uploaded image to insert it at the cursor position in the editor.

### Section, Kind, Contributor and Summary

This collapsible panel contains:

| Field | Description |
|---|---|
| **Section** | Dropdown to assign this content to a section |
| **Kind** | The content type. Options: **Page**, **Article**, **Series** |
| **Contributor** | Dropdown to assign a contributor as the author |
| **Summary** | A brief description used in listings and previews |

### Tags

A text input for adding tags. Tags categorize content across sections.

### Publishing options

| Field | Description |
|---|---|
| **Draft** | When checked, the content is not included in the generated site |
| **Featured** | When checked, the content is marked as featured |
| **Publish Date** | A date and time picker for scheduled publishing. See the [Scheduled Publishing](../scheduling/index.md) guide. |

Click **Save** to create or update the content.

---

## Editing Content

Click **Edit** next to any content item in the list. The edit form is identical to the create form, with these additions:

- The header image can be uploaded and managed
- **Embed** and **Form** toolbar buttons are available
- An autosave indicator in the top-right shows when your changes were last saved (e.g. "Saved just now", "Saved 18s ago")

---

## Content Types

Clio supports three content types:

| Kind | Typical use |
|---|---|
| **Page** | Static pages like "About" or "Contact". Default for new content. |
| **Article** | Blog posts, news items, time-based content |
| **Series** | Multi-part content that belongs to a named series |

The content type affects how it appears in listings and how the generated site organizes it. All three types use the same editor and support the same features.

---

## Draft vs Published

New content is created as a **Draft** by default. Draft content:

- Is visible in the dashboard
- Is not included when the site is generated
- Can be previewed individually

Uncheck the **Draft** checkbox to publish. For published content to appear on the generated site, two conditions must be met:

1. The **Draft** checkbox is unchecked
2. The **Publish Date** is in the past (or empty)

If the publish date is set to a future date, the content is published but will not appear on the site until that date has passed and the site is regenerated. See the [Scheduled Publishing](../scheduling/index.md) guide for details.

---

## Deleting Content

Click **Delete** next to a content item in the list. This removes the content from the database. If the site has already been generated, the previously generated HTML file remains on disk until the site is regenerated.
