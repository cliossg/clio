# Robots.txt

The robots.txt setting lets you control how search engines and web crawlers access your generated site. The content is written as a standard robots.txt file and generated automatically during site build.

## Quick Start

1. Go to **Settings** → **Site** category
2. Find **Robots.txt** and click **Edit**
3. Enter your crawling rules
4. Generate and publish your site

The `robots.txt` file is created at the root of your generated site. The `Sitemap:` directive is appended automatically based on your site's base URL.

## Settings

| Setting | Key | Default | Description |
|---------|-----|---------|-------------|
| Robots.txt | `ssg.robots.txt` | (see below) | Crawling rules for your site |

## Default Content

Clio seeds a default robots.txt that allows all crawlers except AI training bots:

```
User-agent: *
Allow: /

User-agent: GPTBot
Disallow: /

User-agent: ClaudeBot
Disallow: /

User-agent: Google-Extended
Disallow: /
```

You can modify this freely to match your needs.

## How It Works

When the setting has content, Clio writes a `robots.txt` file to the root of the generated site. The `Sitemap:` line is appended automatically using your configured **Site base URL** and **Site base path**, so you don't need to include it manually. If the base URL changes, the sitemap reference updates on the next generation.

A generated robots.txt looks like:

```
User-agent: *
Allow: /

User-agent: GPTBot
Disallow: /

Sitemap: https://yourdomain.com/sitemap.xml
```

If the setting is empty, no robots.txt file is generated.

## Common Rules

### Allow everything

```
User-agent: *
Allow: /
```

### Block all crawlers

```
User-agent: *
Disallow: /
```

### Block specific bots

```
User-agent: GPTBot
Disallow: /

User-agent: ClaudeBot
Disallow: /

User-agent: Google-Extended
Disallow: /

User-agent: CCBot
Disallow: /
```

### Block specific paths

```
User-agent: *
Disallow: /private/
Disallow: /drafts/
```

## Troubleshooting

### robots.txt not appearing on site

- Verify the **Robots.txt** setting is not empty
- Regenerate and republish the site after changing settings

### Sitemap URL is wrong

- Check that **Site base URL** is set correctly in settings
- The Sitemap line is derived from the base URL automatically

### Crawlers ignoring rules

- robots.txt is advisory; well-behaved crawlers follow it but malicious ones may not
- Changes take effect only after crawlers re-read the file (they cache it)
- Verify the file is accessible at `https://yourdomain.com/robots.txt`
