# Clio User Guide

Clio is a static site generator with a web-based content management interface. Write your content, organize it into sections, and generate a fast, static website.

## Clio Running in Five Minutes

You need Docker installed and running. Nothing else.

### 1. Install

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/cliossg/clio/main/deploy/install.sh)
```

This pulls the Docker image, generates a session secret, and starts Clio. It takes about a minute depending on your connection.

### 2. Get your credentials

When the script finishes, it prints your admin email and password. If you missed them:

```bash
cat clio/data/credentials.txt
```

### 3. Sign in

Open `http://localhost:8080` in your browser and log in with those credentials. On first login you will be redirected to a password change form — enter the current password and your new one twice.

### 4. Create your first site

Click **New Site**, give it a name and a slug (e.g., `my-blog`). The slug becomes part of the URL structure.

### 5. Add a section and write something

Create a section (like "Blog" or "Articles"), then add a content entry. Write in the editor or paste Markdown.

### 6. Preview

Click **Preview** in the sidebar. Clio generates your static site and serves it at `http://localhost:3000`.

That's it. You have a working static site generator with a CMS. From here you can [publish to a Git repository](publish/index.md), [set up a public URL with Cloudflare Tunnel](docker/index.md), or keep exploring the features below.

---

## Features

### Content Management

- [**Content**](content/index.md): Create, edit, and manage posts, pages, and articles
- [**Images**](images/index.md): Review and edit metadata for all uploaded images
- [**Import**](import/index.md): Bring Markdown files from your computer into Clio
- [**Proofread**](proofread/index.md): AI-powered grammar, style, and clarity checking
- [**Embeds**](embeds/index.md): Include YouTube, Vimeo, TikTok, and SoundCloud content
- [**Sections**](sections/index.md): Organize content into groups like "Blog", "Docs", or "Tutorials"
- [**Tags**](tags/index.md): Cross-cutting labels for categorizing content across sections

### Site Management

- [**Preview**](preview/index.md): Generate and preview your site locally before publishing
- [**Publish**](publish/index.md): Deploy your generated site to a Git repository
- [**Backup and Restore**](backup/index.md): Export your site and restore it on another instance
- [**Scheduled Publishing**](scheduling/index.md): Publish content automatically at a future date and time
- [**Google Analytics**](analytics/index.md): Track visitor traffic with Google Analytics
- [**Cookie Banner**](cookie-banner/index.md): Cookie consent banner for your site
- [**Google Search**](search/index.md): Add site search using Google Programmable Search Engine
- [**Robots.txt**](robots-txt/index.md): Control how search engines and crawlers access your site
- [**Layouts**](layouts/index.md): Create custom templates that control how your content is rendered
- [**Settings**](settings/index.md): System and user-defined configuration for your site

### Interaction

- [**Contact Forms**](forms/index.md): Receive messages from visitors via a contact form
- [**Contributors**](contributors/index.md): Manage content authors and their public profiles
- [**REST API**](api/index.md): Manage content programmatically with the local REST API

### Getting Started

- [**Installation**](install/index.md): Install Clio with a single command, clone and run from source, or build a native binary
- [**Sites**](sites/index.md): Create, select, and manage your sites
- [**Site Dashboard**](sites/dashboard/index.md): Navigate a selected site and its features

### Deployment

- [**Self-Hosting with Docker**](docker/index.md): Run Clio with Docker and expose your site with Cloudflare Tunnel


## Quick Overview

### How Clio Works

1. **Create a site** with a name and slug
2. **Organize with sections** (like "Blog", "Docs", "About")
3. **Write content** in the web editor or import Markdown files
4. **Generate** your static site
5. **Publish** to your hosting provider

### Key Concepts

| Concept     | Description                                                             |
| ----------- | ----------------------------------------------------------------------- |
| **Site**    | A website project with its own content, settings, and generated output  |
| **Section** | A category or folder for organizing content (e.g., "Blog", "Tutorials") |
| **Content** | An individual page or article written in Markdown                       |
| **Layout**  | A template that controls how content is rendered                        |
| **Tag**     | A label for cross-cutting categorization                                |

## Getting Help

Each feature guide includes:
- A quick start for common tasks
- Detailed explanations of concepts
- Step-by-step workflows
- Troubleshooting tips

Start with the feature you need, or read through them all to understand Clio's full capabilities.
