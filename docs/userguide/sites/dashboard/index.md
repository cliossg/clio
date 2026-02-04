# Site Dashboard

When you select a site from the [sites list](../index.md), you enter the site dashboard. This is the home page for a specific site and the hub for all actions within it.

The dashboard shows the site name, its properties, and groups of action cards that link to each feature.

## Navigation Bar

Inside a site, the navigation bar switches to site-specific sections:

| Link | What it does |
|---|---|
| **Sites** | Return to the [sites list](../index.md) |
| **Layouts** | Templates that control how content is rendered |
| **Messages** | Contact form submissions from visitors |
| **Contributors** | People who create content for this site |
| **Settings** | Key-value configuration for this site |

All links operate within the selected site. Clicking **Messages** shows messages for this site only.

The **Sites** link is the only item that appears in both navigations. It always takes you back to the sites list.

## Breadcrumbs

Every page inside a site shows a breadcrumb link at the top that takes you one level up:

- From a content list or settings page: **← Site Name** goes back to this dashboard
- From this dashboard: **← Sites** goes back to the sites list

## Properties

The top of the dashboard shows:

- **Site name** as the page title
- **Slug**: the URL-friendly identifier
- **Status**: Active or inactive (shown as a badge)
- **Created** and **Updated**: timestamps
- **Edit** and **Delete** buttons in the top-right corner

Clicking **Edit** opens the [edit form](../index.md#editing-a-site). Clicking **Delete** removes the site from the database (generated files on disk are kept).

---

## Action Cards

Below the properties, the dashboard organizes features into four groups.

### Content

| Card | What it does |
|---|---|
| **Content** | Manage posts, pages, and articles for this site. See the [Content](../../content/index.md) guide. |
| **Images** | Review and edit metadata for all uploaded images. See the [Images](../../images/index.md) guide. |
| **Import** | Import Markdown files from your computer. See the [Import](../../import/index.md) guide. |

### Organization

| Card | What it does |
|---|---|
| **Sections** | Organize content into sections (e.g. "Blog", "Docs", "Tutorials"). See the [Sections](../../sections/index.md) guide. |
| **Tags** | Categorize content with labels that work across sections. See the [Tags](../../tags/index.md) guide. |

### Markdown

| Card | What it does |
|---|---|
| **Backup** | Export content, metadata, and images to a Git repository. See the [Backup and Restore](../../backup/index.md) guide. |
| **Restore** | Recreate a site from a backup directory. See the [Backup and Restore](../../backup/index.md) guide. |

### Publish

| Card | What it does |
|---|---|
| **Preview** | Generate the static site and serve it on the preview server (`localhost:3000`). See the [Preview](../../preview/index.md) guide. |
| **Publish** | Deploy the generated site to a Git repository. See the [Publish](../../publish/index.md) guide. |

---

## Cards vs Navigation

Some features are accessible from both the dashboard cards and the navigation bar. Others are only in one place.

| Feature | Dashboard card | Navigation bar |
|---|---|---|
| Content | Yes | |
| Images | Yes | |
| Import | Yes | |
| Sections | Yes | |
| Tags | Yes | |
| Backup | Yes | |
| Restore | Yes | |
| Preview | Yes | |
| Publish | Yes | |
| Layouts | | Yes |
| Messages | | Yes |
| Contributors | | Yes |
| Settings | | Yes |

The dashboard cards give quick access to features that are specific to content management and publishing. The navigation bar provides access to site-wide configuration (layouts, settings) and auxiliary features (messages, contributors).

---

## Layouts

The Layouts page lists all templates available for this site. Each layout controls how content is rendered on the generated site: its HTML structure, CSS styles, and overall visual design.

The list shows each layout's name and description. From here you can create a new layout or edit an existing one.

A site can have multiple layouts. You assign a default layout to the site (from the [edit form](../index.md#editing-a-site)) and can override it per section. Pages in a section without a specific layout use the site default. If no layouts exist, Clio uses a built-in embedded layout.

See the [Layouts](../../layouts/index.md) guide for details on creating templates, available data, and styling.

---

## Messages

The Messages page shows contact form submissions received through your published site. Each message displays the sender's name, email, a preview of the message body, the date it was received, and its status.

Messages can be marked as **read** or **unread**. This page is where you review and manage incoming communication from visitors. For details on how to add a contact form to your site, see the [Contact Forms](../../forms/index.md) guide.

---

## Contributors

The Contributors page lists the people who create content for this site. Each contributor has a **handle** (e.g. `@johndoe`) and a **name**.

From here you can create new contributors, edit their details, or view their profile. Contributors are assigned to content items to indicate authorship. A site can have multiple contributors, and each contributor belongs to a single site.

See the [Contributors](../../contributors/index.md) guide for details on managing contributors and their public profiles.

---

## Settings

The Settings page shows all key-value configuration entries for this site. Each setting has a **name**, a **value**, and a **type** (string, boolean, integer, or text).

Settings control various behaviors across Clio features. For example, backup repository URLs, analytics tracking IDs, display options, and cookie banner preferences are all stored as settings. You can create new settings or edit existing ones from this page.

See the [Settings](../../settings/index.md) guide for the full list of system settings, data types, and details on creating user-defined settings.
