# Layouts

Layouts are templates that control how your content is rendered on the generated site. Each layout defines the HTML structure, visual design, and behavior of every page. Click **Layouts** in the navigation bar to manage layouts for the current site.

## The Layouts List

The list shows all layouts for the current site. Each row displays the layout name, description, and an Edit button. Click **New Layout** to create one.

## Creating and Editing a Layout

The layout form has the following fields:

| Field | Description |
|---|---|
| **Name** | A name for this layout (e.g. "Magazine", "Minimal") |
| **Description** | A short description of the layout's visual style |
| **Layout Code (HTML/Template)** | The full HTML template using Go template syntax |
| **Custom CSS** | CSS styles injected into pages that use this layout |
| **Exclude default Clio CSS** | When checked, the default theme stylesheet is not loaded. Only `core.css` and your custom CSS are used. |

Click **Update Layout** to save or **Cancel** to discard.

---

## How Layouts Work

A layout is a complete HTML page written in Go's template syntax. When Clio generates your site, it passes page data to the layout and renders the final HTML.

The layout must be defined as a named template called `layout.html`:

```html
{{ define "layout.html" }}
<!DOCTYPE html>
<html lang="en">
<head>
    <title>{{ .Site.Name }}</title>
</head>
<body>
    <!-- your page structure here -->
</body>
</html>
{{ end }}
```

If no custom layout is assigned, Clio uses a built-in default template.

### Layout Resolution

Clio selects a layout in this order:

1. The layout assigned to the content's section (if set)
2. The site's default layout (set from the [site edit form](../sites/index.md#editing-a-site))
3. The built-in embedded template

This means you can use different layouts for different sections. A blog section can have a magazine-style layout while a documentation section uses a minimal one.

---

## Template Blocks

A single layout template handles all page types: index pages, content detail pages, author pages, and search. The layout uses conditional logic to render the right content based on the page type.

The key flags that distinguish page types:

| Flag | True when rendering |
|---|---|
| `.IsIndex` | An index or listing page (home page, section index) |
| `.IsAuthor` | An author profile page |
| `.IsSearch` | The search results page |
| (none of the above) | A single content page (article, post, page) |

A typical layout routes to different sections based on these flags:

```html
{{ define "layout.html" }}
<!DOCTYPE html>
<html lang="en">
<head>
    <title>
        {{ if .IsIndex }}{{ .Site.Name }}
        {{ else if .IsAuthor }}@{{ .Author.Handle }} - {{ .Site.Name }}
        {{ else }}{{ .Content.Heading }} - {{ .Site.Name }}
        {{ end }}
    </title>
</head>
<body>
    <nav>
        <a href="{{ .AssetPath }}">{{ .Site.Name }}</a>
        {{ range .Menu }}
        <a href="{{ .AssetPath }}{{ .Path }}/">{{ .Name }}</a>
        {{ end }}
    </nav>

    {{ if .IsSearch }}
        <!-- search page content -->
    {{ else if .IsAuthor }}
        <!-- author profile -->
    {{ else if .IsIndex }}
        <!-- content listing -->
    {{ else }}
        <!-- single content page -->
    {{ end }}
</body>
</html>
{{ end }}
```

---

## Available Data

All page types receive the same data structure. Some fields are only populated for certain page types.

### Common Fields (All Pages)

| Field | Type | Description |
|---|---|---|
| `.Site.Name` | string | The site name |
| `.Site.Slug` | string | The site slug |
| `.Menu` | list | Sections available for navigation |
| `.Section` | object | The current section (if applicable) |
| `.Sections` | list | All sections |
| `.AssetPath` | string | Base URL path (e.g. `/` or `/blog/`) |
| `.Params` | map | All site settings as key-value pairs |
| `.CustomCSS` | string | CSS from the layout's Custom CSS field |
| `.ExcludeDefaultCSS` | bool | Whether to skip the default theme stylesheet |

Access site settings with `{{ index .Params "setting.key" }}`. For example: `{{ index .Params "ssg.analytics.id" }}`.

### Menu Items

Each item in `.Menu` is a section:

| Field | Type | Description |
|---|---|---|
| `.Name` | string | Section name |
| `.Path` | string | URL path (e.g. `blog`) |
| `.Description` | string | Section description |

### Index Pages (`.IsIndex` is true)

| Field | Type | Description |
|---|---|---|
| `.Contents` | list | Content items for the current page |
| `.IsPaginated` | bool | Whether there are multiple pages |
| `.CurrentPage` | int | Current page number (starts at 1) |
| `.TotalPages` | int | Total number of pages |
| `.HasPrev` | bool | Whether a previous page exists |
| `.HasNext` | bool | Whether a next page exists |
| `.PrevURL` | string | URL to the previous page |
| `.NextURL` | string | URL to the next page |
| `.Section` | object | The section being listed (includes `.Section.HeaderImageURL`, `.Section.HeroTitleDark`, `.Section.Description`) |

Each item in `.Contents` is a rendered content object (see Content Fields below).

### Content Pages (not index, not author, not search)

| Field | Type | Description |
|---|---|---|
| `.Content` | object | The content being displayed (see Content Fields below) |
| `.Blocks` | object | Related content and series navigation (see Blocks below) |

### Author Pages (`.IsAuthor` is true)

| Field | Type | Description |
|---|---|---|
| `.Author.Name` | string | First name |
| `.Author.Surname` | string | Last name |
| `.Author.Handle` | string | Handle without the `@` |
| `.Author.Bio` | string | Biography text |
| `.Author.PhotoPath` | string | Relative path to profile photo |
| `.Author.SocialLinks` | list | Social media links (each has `.Platform`, `.URL`) |
| `.Contents` | list | All content by this author |

---

## Content Fields

These fields are available on `.Content` (detail pages) and on each item when iterating `.Contents` (index pages, author pages, blocks).

| Field | Type | Description |
|---|---|---|
| `.Heading` | string | The title |
| `.Summary` | string | Short description |
| `.HTMLBody` | HTML | Rendered Markdown as HTML |
| `.URL` | string | Full URL path to this content |
| `.Kind` | string | Content type: `page`, `article`, `series` |
| `.Draft` | bool | Whether this is a draft |
| `.Featured` | bool | Whether this is featured |
| `.PublishedAt` | time | Publication date (use `.PublishedAt.Format "January 2, 2006"`) |
| `.SectionName` | string | Name of the assigned section |
| `.SectionPath` | string | URL path of the section |
| `.Series` | string | Series name (if part of a series) |
| `.SeriesOrder` | int | Position within the series |
| `.Tags` | list | Tags (each has `.Name` and `.Slug`) |
| `.ContributorHandle` | string | Contributor's handle (without `@`) |
| `.AuthorUsername` | string | Author's username (fallback if no contributor) |
| `.HeaderImageURL` | string | URL to the header/hero image |
| `.HeaderImageAlt` | string | Alt text for the header image |
| `.HeaderImageCaption` | string | Image caption |
| `.HeaderImageAttribution` | string | Image credit text |
| `.HeaderImageAttributionURL` | string | Link to the image source |
| `.HeroTitleDark` | bool | Whether the hero overlay should use dark text |
| `.Meta` | object | SEO metadata (see below) |

### SEO Metadata (`.Content.Meta`)

| Field | Type | Description |
|---|---|---|
| `.Description` | string | Meta description |
| `.Keywords` | string | Meta keywords |
| `.CanonicalURL` | string | Canonical URL |
| `.Robots` | string | Robots directive (e.g. `noindex`) |
| `.TableOfContents` | bool | Whether to show a table of contents |

---

## Related Content Blocks

On content detail pages, `.Blocks` provides related content and series navigation. Check `.Blocks.HasContent` before rendering.

### Related Articles

For non-series content, Clio finds articles with matching tags:

| Field | Type | Description |
|---|---|---|
| `.Blocks.Related` | list | Related content items (each is a rendered content object with `.Heading`, `.URL`, etc.) |

### Series Navigation

For content that belongs to a series:

| Field | Type | Description |
|---|---|---|
| `.Blocks.SeriesPrev` | object | Previous item in the series (or nil) |
| `.Blocks.SeriesNext` | object | Next item in the series (or nil) |
| `.Blocks.SeriesIndexBackward` | list | All items before the current one in the series |
| `.Blocks.SeriesIndexForward` | list | All items after the current one in the series |

Example usage:

```html
{{ if and .Blocks .Blocks.HasContent }}
<aside>
    {{ if .Content.Series }}
        <h3>Series: {{ .Content.Series }}</h3>
        {{ if .Blocks.SeriesPrev }}
        <a href="{{ .Blocks.SeriesPrev.URL }}">← {{ .Blocks.SeriesPrev.Heading }}</a>
        {{ end }}
        {{ if .Blocks.SeriesNext }}
        <a href="{{ .Blocks.SeriesNext.URL }}">{{ .Blocks.SeriesNext.Heading }} →</a>
        {{ end }}
    {{ else }}
        <h3>Related</h3>
        <ul>
        {{ range .Blocks.Related }}
            <li><a href="{{ .URL }}">{{ .Heading }}</a></li>
        {{ end }}
        </ul>
    {{ end }}
</aside>
{{ end }}
```

---

## Styles

Layouts control styling through two mechanisms:

### Custom CSS

The **Custom CSS** field on the layout form lets you write CSS that is injected into a `<style>` tag in the page head. This CSS applies to all pages that use this layout.

If you are writing a custom layout from scratch, you can put all your styles here and check **Exclude default Clio CSS** to prevent the default theme from loading. Only `core.css` (basic resets and structural styles) is always included.

If you want to build on top of the default theme, leave the checkbox unchecked and use Custom CSS to override specific styles.

### Inline Styles in the Template

You can also include `<link>` tags or `<style>` blocks directly in your layout code to load external stylesheets (e.g. Google Fonts) or define styles inline.

---

## Template Functions

In addition to Go's built-in template functions, these helpers are available:

### String

`upper`, `lower`, `title`, `truncate`, `contains`, `hasPrefix`, `hasSuffix`, `replace`, `split`, `join`

### Math

`add`, `sub`, `mul`, `div`, `seq`

### HTML

`safeHTML`, `safeAttr`, `safeURL`

### Other

`now` (returns the current time), `len`, `first`, `last`, `eq`, `ne`, `lt`, `le`, `gt`, `ge`

Example: `{{ .Content.Heading | truncate 50 }}`, `{{ now.Year }}`, `{{ len .Contents }}`

---

## Quick Start

To create a layout from scratch:

1. Go to **Layouts** → **New Layout**
2. Give it a name and description
3. In the Layout Code field, start with `{{ define "layout.html" }}` and end with `{{ end }}`
4. Write your HTML structure, using the template variables documented above
5. Add your CSS in the Custom CSS field
6. Check **Exclude default Clio CSS** if you want full control over styling
7. Save the layout
8. Assign it as the site default (from the [site edit form](../sites/index.md#editing-a-site)) or to a specific section (from the [section edit form](../sections/index.md))
9. Click **Preview** to see the result
