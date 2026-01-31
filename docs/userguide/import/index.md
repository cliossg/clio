# Import

Import lets you bring Markdown files from your computer into Clio without moving them. Your files stay where they are. Clio just tracks and syncs them.

## Why use Import?

- **Write anywhere**: Use your favorite editor (Vim, VS Code, Obsidian)
- **Sync with cloud storage**: Keep files in Dropbox, iCloud, or Google Drive
- **Batch content creation**: Import multiple articles at once
- **Version control friendly**: Keep your content in a Git repository

## Quick Start

1. Create a Markdown file in your import directory:
   ```
   ~/Documents/Clio/your-site-slug/my-article.md
   ```

2. Go to your site in Clio and click **Import** in the navigation

3. Select your file, choose a target section, and click **Import Selected**

That's it. Your content is now in Clio.

## The Import Directory

Each site has its own import directory based on its slug:

```
~/Documents/Clio/
├── my-blog/
│   ├── first-post.md
│   └── second-post.md
├── docs-site/
│   └── getting-started.md
```

When you create a new site, Clio offers to create this directory for you.

> **Tip**: You can change the base path from `~/Documents` to another location (like `~/Dropbox`) using the `import.base-path` setting.

## File Status

Every file in the import list has a status badge:

| Badge                 | Meaning                                           | What you can do                                   |
| --------------------- | ------------------------------------------------- | ------------------------------------------------- |
| **New** (blue)        | File exists but hasn't been imported yet          | Select and import                                 |
| **Synced** (gray)     | File was imported and nothing changed             | View the content                                  |
| **Reimport** (yellow) | You edited the file after importing               | Select to update the content                      |
| **Conflict** (red)    | Both the file AND the web content were edited     | Select to force reimport (overwrites web changes) |
| **Missing** (gray)    | The file was deleted but the content still exists | View the content                                  |

### Understanding Synced vs Reimport

When you import a file, Clio records the file's modification time. Later:

- If the file hasn't changed → **Synced**
- If you edit the file (in Vim, etc.) → **Reimport**

Synced files appear grayed out since no action is needed.

### Understanding Conflicts

A conflict happens when:

1. You import a file
2. You edit the content in Clio's web interface
3. You also edit the original file

Both versions now have changes. Selecting a conflicted file and clicking "Import Selected" will overwrite the web changes with the file content.

## Common Workflows

### Writing in your editor, publishing in Clio

1. Write your article in `~/Documents/Clio/my-site/new-post.md`
2. Open Clio → Import → Select the file → Choose section → Import
3. Edit metadata (tags, images) in Clio if needed
4. Publish

### Updating an existing article

1. Edit the file in your editor
2. Open Clio → Import → The file now shows "Reimport"
3. Select it → Click "Import Selected"
4. The content in Clio is updated

### Importing multiple files at once

1. Place several `.md` files in your import directory
2. Open Clio → Import
3. Check all the files you want (or use the header checkbox)
4. Choose a target section
5. Click "Import Selected"

All selected files are imported in one operation.

## Frontmatter

You can add YAML frontmatter to control how your content is imported:

```yaml
---
title: "My Article Title"
draft: false
summary: "A brief description"
---

Your content here...
```

### Supported fields

| Field         | Description                                         |
| ------------- | --------------------------------------------------- |
| `title`       | Content title (otherwise uses first H1 or filename) |
| `draft`       | `true` or `false` (default: true)                   |
| `summary`     | Short description                                   |
| `author`      | Username of the author                              |
| `contributor` | Handle of the contributor                           |
| `kind`        | `page`, `article`, or `series`                      |
| `series`      | Series name (for multi-part content)                |
| `featured`    | `true` to mark as featured                          |

### SEO fields

| Field           | Description                         |
| --------------- | ----------------------------------- |
| `description`   | Meta description for search engines |
| `keywords`      | Comma-separated keywords            |
| `robots`        | e.g., `noindex`                     |
| `canonical-url` | Canonical URL if cross-posting      |

### Title resolution

If you don't specify a `title` in frontmatter, Clio looks for the first `# Heading` in your content. If there's no H1 either, it uses the filename.

## Filtering the List

Use the filter tabs above the table to focus on specific files:

- **All**: Shows everything
- **New**: Only files that haven't been imported
- **Updated**: Files ready for reimport
- **Conflicts**: Files with conflicts to resolve

## Configuration

### Changing the import directory

By default, Clio looks for files in `~/Documents/Clio/{site-slug}/`.

To use a different base path (e.g., Dropbox):

1. Go to **Settings** for your site
2. Add a setting with key `import.base-path`
3. Set the value to your preferred path (e.g., `~/Dropbox`)

Your import directory becomes `~/Dropbox/Clio/{site-slug}/`.

## Troubleshooting

### File doesn't appear in the import list

- Check that the file is in the correct directory: `~/Documents/Clio/{your-site-slug}/`
- Make sure the file has a `.md` extension
- Verify the site slug matches the directory name

### Import says "Conflict" but I didn't edit in the web UI

This can happen if:
- The content was auto-saved in Clio
- Someone else edited the content
- The timestamps got out of sync

Solution: If you're sure the file version is correct, select it and import anyway. This overwrites the web version.

### Changes in the file aren't detected

Clio checks the file's modification time. Some editors or sync tools preserve the original timestamp. Try:

```bash
touch ~/Documents/Clio/my-site/my-article.md
```

Then refresh the import page.
