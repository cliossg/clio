# Sites

Sites is the first page you see after signing in. It lists all the sites you have created and is the starting point for everything you do in Clio.

From here you can:

- Create a new site
- Select a site to work on
- Edit or delete existing sites

## Navigation Bar

When no site is selected, the navigation bar shows sections that apply to the whole Clio instance:

| Link | What it does |
|---|---|
| **Sites** | The sites list (this page) |
| **API** | API token management |
| **Users** | User accounts |
| **Profile** | Your own account settings |

Your username and a **Sign Out** link are always visible on the right side.

Once you select a site, the navigation changes. See the [Site Dashboard](dashboard/index.md) guide for details.

---

## Creating a Site

Click **New Site** in the top-right corner. The form has three fields:

| Field | Description |
|---|---|
| **Name** | The display name for your site (e.g. "My Blog") |
| **Slug** | The URL-friendly identifier (e.g. `my-blog`). Used in paths and directory names. |
| **Create import directory** | When checked, creates `~/Documents/Clio/{slug}/` for importing Markdown files. Enabled by default. |

Click **Create** and your site appears in the list.

### What is the slug?

The slug is a short, lowercase identifier with no spaces. It is used as:

- The directory name for generated site files
- The import directory name (`~/Documents/Clio/{slug}/`)
- Part of internal URLs

Choose something short and descriptive. You can change it later from the Edit page, but the import directory path will change with it.

---

## Selecting a Site

Click a site name in the list to enter it. This does two things:

1. Opens the [site dashboard](dashboard/index.md), which shows all available actions organized into cards
2. Stores the selected site in your browser so Clio remembers it between visits

If you close the browser and come back, Clio redirects you to the last site you were working on.

### Switching to another site

Click **Sites** in the navigation bar, or click the **← Sites** breadcrumb link at the top of the site dashboard. This returns you to the sites list where you can select a different site.

---

## Editing a Site

Click **Edit** next to any site in the list, or click the **Edit** button on the site dashboard. The edit form lets you change:

| Field | Description |
|---|---|
| **Name** | The display name |
| **Slug** | The URL identifier |
| **Active** | Whether the site is active |
| **Default Layout** | The layout applied to pages that don't have a section-specific layout |

Click **Save** to apply changes.

---

## Deleting a Site

Click **Delete** next to a site in the list, or from the Edit page.

Deleting a site removes it from the database: content, sections, tags, contributors, settings, and all associated records are deleted. The generated files on disk are not removed. This is intentional. If you need the files cleaned up, delete the site's directory manually from the data folder.

---

## Multiple Sites

Clio supports multiple independent sites from a single installation. Each site has its own:

- Content and sections
- Layouts and templates
- Tags and contributors
- Images and media
- Settings and configuration
- Generated output

Sites do not share data. Creating a section in one site does not affect another.

This is useful if you maintain several projects. A personal blog and a documentation site can coexist in the same Clio instance without interfering with each other.
