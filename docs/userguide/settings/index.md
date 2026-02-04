# Settings

Settings are key-value configuration entries that control how your site is generated and published. Click **Settings** in the navigation bar to manage settings for the current site.

## The Settings List

The list shows all settings for the current site, ordered by category and position. Each row displays:

| Column | Description |
|---|---|
| **Name** | The setting name |
| **Value** | The current value (sensitive values like tokens are masked) |
| **Type** | The data type (string, boolean, integer, text) |
| **Actions** | Edit button |

---

## System vs User Settings

Settings are either **system** or **user-defined**.

### System Settings

System settings are created automatically when a site is created. They configure Clio's built-in features: analytics, publishing, forms, search, display options, and more. Each system setting has a name, description, type, and default value defined by Clio.

When editing a system setting, only the **value** can be changed. The name, description, and reference key are locked.

### User Settings

User settings are created manually with the **New Setting** button. All fields (name, description, value) are editable.

User-defined settings are currently stored but not yet accessible in templates. A future version of Clio will make them available via the `.Params` map in [layouts](../layouts/index.md), enabling use cases like dynamic layout customization without modifying template code.

---

## Creating a Setting

Click **New Setting** in the top-right corner. The form has the following fields:

| Field | Description |
|---|---|
| **Name** | A descriptive name (e.g. `site_title`, `analytics_id`) |
| **Description** | What this setting is used for |
| **Value** | The setting's value |
| **Reference Key** | Optional. An internal key for programmatic access to this setting. |

Click **Create Setting** to save or **Cancel** to discard.

---

## Editing a Setting

Click **Edit** next to a setting in the list. For system settings, the form shows the name and description as read-only, with the message "System setting - only value can be modified." For user settings, all fields except the reference key are editable.

The value field renders differently depending on the setting type:

| Type | UI Control | Description |
|---|---|---|
| **boolean** | Checkbox toggle | On/off switch |
| **string** | Text input | Single-line text |
| **text** | Textarea | Multi-line text |
| **integer** | Number input | Numeric value, may have min/max constraints |

---

## Default System Settings

When you create a site, Clio seeds the following settings with default values. They are grouped by category.

### Site

| Setting | Description | Default |
|---|---|---|
| **Site description** | Description shown in hero and meta tags | |
| **Hero image** | Hero image filename for the site index | |
| **Site base path** | Base path for GitHub Pages subpath hosting | `/` |
| **Site base URL** | Full base URL (e.g. `https://example.com`) | `https://example.com` |
| **Cookie banner enabled** | Show cookie consent banner | `true` |
| **Cookie banner text** | Cookie banner consent message | (default message) |
| **Robots.txt** | Custom robots.txt content (sitemap URL is appended automatically) | (default rules) |

### Display

| Setting | Description | Default |
|---|---|---|
| **Index max items** | Maximum items shown on index pages | `9` |
| **Blocks enabled** | Show related content blocks on content pages | `true` |
| **Blocks max items** | Maximum items in a related content block | `5` |
| **Blocks multi-section** | Include related content from other sections | `true` |
| **Blocks background color** | Background color for related content blocks | `#f0f4f8` |

### Analytics

| Setting | Description | Default |
|---|---|---|
| **Google Analytics enabled** | Enable Google Analytics tracking | `true` |
| **Google Analytics ID** | Google Analytics measurement ID (e.g. `G-XXXXXXXXXX`) | |

### Search

| Setting | Description | Default |
|---|---|---|
| **Google Search enabled** | Enable Google site search | `true` |
| **Google Search ID** | Google Custom Search Engine ID | |

### Git

| Setting | Description | Default |
|---|---|---|
| **Publish repository URL** | Git repository URL for publishing | |
| **Publish branch** | Git branch for publishing | `gh-pages` |
| **Publish auth token** | Authentication token for publishing | |
| **Backup repository URL** | Git repository URL for markdown backup | |
| **Backup branch** | Git branch for markdown backup | `main` |
| **Backup auth token** | Authentication token for backup | |
| **Commit user name** | Git user name for commits | `Clio Bot` |
| **Commit user email** | Git user email for commits | `clio@localhost` |

### Scheduling

| Setting | Description | Default |
|---|---|---|
| **Scheduled publish enabled** | Enable automatic publishing of scheduled content | `true` |
| **Scheduled publish interval** | How often to check for scheduled content (e.g. `1h`, `30m`) | `15m` |

### API

| Setting | Description | Default |
|---|---|---|
| **API enabled** | Enable the REST API for external clients | `false` |

### Forms

| Setting | Description | Default |
|---|---|---|
| **Forms enabled** | Enable contact form submissions | `false` |
| **Forms endpoint URL** | Public URL where the forms server is reachable | |
| **Forms allowed origins** | Comma-separated list of allowed origins for CORS | |
| **Forms rate limit** | Maximum form submissions per IP per hour | `5` |

---

## Settings in Other Guides

Many system settings are documented in detail in their respective feature guides:

- Git settings: [Publish](../publish/index.md) and [Backup and Restore](../backup/index.md)
- Display settings: [Preview](../preview/index.md)
- Scheduling settings: [Scheduled Publishing](../scheduling/index.md)
- Analytics settings: [Google Analytics](../analytics/index.md)
- Cookie banner settings: [Cookie Banner](../cookie-banner/index.md)
- Search settings: [Google Search](../search/index.md)
- Forms settings: [Contact Forms](../forms/index.md)
- API settings: [REST API](../api/index.md)
- Robots.txt: [Robots.txt](../robots-txt/index.md)
- Import base path: [Import](../import/index.md)

---

## Deleting a Setting

Click **Delete** on the setting detail or edit page. Both system and user settings can be deleted. If you delete a system setting, it will not be re-created automatically. To restore it, you would need to create it manually with the correct reference key.
