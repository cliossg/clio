# Backup and Restore

Backup and Restore lets you export your entire Clio site to a portable format and restore it on another instance. This is useful for migrations, disaster recovery, or maintaining offline backups.

## Why use Backup and Restore?

- **Migration**: Move your site to a new server or Clio instance
- **Disaster recovery**: Restore from backup if something goes wrong
- **Version control**: Keep your site content in a Git repository
- **Portability**: Your content stays yours, in standard formats

## Quick Start

### Backup

1. Go to your site in Clio
2. Navigate to **Settings** → **Publishing**
3. Configure a backup repository (Git URL + token)
4. Click **Backup**

Your site is exported to the repository as Markdown files with metadata.

### Restore

1. On a new or existing Clio instance, go to your site
2. Navigate to **Restore**
3. Enter the path to your backup directory
4. Click **Restore**

Your content, sections, tags, contributors, and images are recreated.

## Backup Structure

When you backup a site, Clio creates this structure:

```
backup/
├── content/
│   ├── blog/
│   │   ├── first-post.md
│   │   └── second-post.md
│   └── docs/
│       └── getting-started.md
├── meta/
│   ├── layouts.yml
│   ├── layouts/
│   │   ├── Default.html
│   │   └── Default.css
│   ├── sections.yml
│   ├── contributors.yml
│   ├── tags.yml
│   ├── images.yml
│   └── content_images.yml
├── images/
│   ├── blog/
│   │   └── hero.jpg
│   └── shared/
│       └── logo.png
└── profiles/
    └── johndoe.jpg
```

### Content Files

Each content item becomes a Markdown file with YAML frontmatter:

```yaml
---
title: "My First Post"
slug: "my-first-post-a1b2c3d4"
section: "blog"
draft: false
featured: true
contributor: "johndoe"
tags:
  - "tutorial"
  - "getting-started"
published_at: "2024-01-15T10:30:00Z"
description: "Learn the basics"
keywords: "tutorial, basics, intro"
---

Your content here...
```

### Meta Files

The `meta/` directory contains YAML files describing your site structure:

| File | Contents |
|------|----------|
| `layouts.yml` | Layout names and settings |
| `layouts/*.html` | Layout template code |
| `layouts/*.css` | Layout custom CSS |
| `sections.yml` | Section names, paths, and assigned layouts |
| `contributors.yml` | Contributor profiles (handle, name, bio, social links) |
| `tags.yml` | Tag names and slugs |
| `images.yml` | Image metadata (alt text, captions, attribution) |
| `content_images.yml` | Content-to-image relationships |

## Rich vs Basic Restore

Clio supports two restore modes depending on what's available:

### Rich Restore

When the backup includes `meta/` directory:

- Layouts are recreated with their templates and CSS
- Sections are recreated with their layouts
- Contributors are recreated with social links
- Tags are recreated
- Images are copied and metadata is applied
- Content is imported with all relationships intact

This is the default when restoring from a Clio backup.

### Basic Restore

When restoring plain Markdown files without `meta/`:

- Content is imported based on frontmatter
- Section is resolved from the `section` field in frontmatter
- Contributor is resolved from the `contributor` field
- Tags are resolved from the `tags` array
- Missing sections/contributors/tags are created automatically

This works for importing from external sources or older backups.

## Common Workflows

### Migrating to a new server

1. On the old server, configure backup to a Git repository
2. Run **Backup**
3. On the new server, create a site with the same slug
4. Clone the backup repository locally
5. Run **Restore** pointing to the local path

### Regular backups to Git

1. Configure a private Git repository for backups
2. Periodically click **Backup**
3. Each backup creates a new commit with timestamp
4. Your content history is preserved in Git

### Restoring a specific version

1. Clone your backup repository
2. Check out the desired commit
3. Run **Restore** pointing to that directory

## Frontmatter Reference

The backup includes all content metadata in frontmatter:

### Basic fields

| Field | Description |
|-------|-------------|
| `title` | Content title |
| `slug` | URL slug (includes short ID) |
| `section` | Section path (e.g., "blog") |
| `draft` | Publication status |
| `featured` | Featured flag |
| `kind` | Content type: page, article, series |
| `series` | Series name (for multi-part content) |
| `series_order` | Position in series |

### Attribution

| Field | Description |
|-------|-------------|
| `author` | Original author username |
| `contributor` | Contributor handle |

### SEO

| Field | Description |
|-------|-------------|
| `description` | Meta description |
| `keywords` | Meta keywords |
| `robots` | Robots directive |
| `canonical_url` | Canonical URL |

### Dates

| Field | Description |
|-------|-------------|
| `published_at` | Publication date (ISO 8601) |
| `created_at` | Creation date |
| `updated_at` | Last update date |

### Tags

```yaml
tags:
  - "first-tag"
  - "second-tag"
```

## Configuration

### Backup repository

Configure in **Settings** → **Publishing**:

| Setting | Description |
|---------|-------------|
| `publish.repo-url` | Git repository URL |
| `publish.backup-branch` | Branch for backups (default: backup) |
| `publish.auth-token` | Authentication token |

### Restore path

The restore path can be:

- An absolute path: `/home/user/backup`
- A home-relative path: `~/backups/my-site`
- A path to a cloned Git repository

## Troubleshooting

### Restore fails with "Directory does not exist"

Verify the path is correct and accessible by the Clio process.

### Missing images after restore

Images are only restored if the `images/` directory exists in the backup. Make sure your backup includes the images folder.

### Contributors not linked to profiles

Profile photos are stored in `profiles/` and linked separately. If contributor photos are missing, check that:

1. The `profiles/` directory exists in the backup
2. Photo filenames match contributor handles

### Duplicate content after restore

Restore is idempotent for metadata (sections, tags, contributors) but creates new content entries. To avoid duplicates, restore to a fresh site or delete existing content first.
