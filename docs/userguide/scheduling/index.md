# Scheduled Publishing

Scheduled publishing lets you write content now and have it go live at a specific date and time. Clio checks for pending content on a regular interval and publishes automatically.

## Why use Scheduled Publishing?

- **Plan ahead**: Write a week of posts and schedule them in advance
- **Consistent cadence**: Publish at the same time every week without manual work
- **Time zones**: Schedule for when your audience is active
- **Batch workflow**: Write when inspired, publish when it matters

## Quick Start

1. Create or edit a content item
2. Set **Published At** to a future date and time
3. Make sure the content is **not a draft**
4. Enable scheduled publishing in your site settings

When the scheduled time arrives, Clio publishes the content automatically on the next check.

## How It Works

The scheduler runs in the background while Clio is running. On each interval:

1. It checks all sites that have scheduling enabled
2. For each site, it looks for content that meets all three conditions:
   - Not a draft
   - Has a `Published At` date in the past (the scheduled time has arrived)
   - Was published after the last automatic publish
3. If pending content is found, it generates the site and publishes to git

Content with a future `Published At` date is excluded from site generation entirely. It won't appear on index pages, feeds, or anywhere on the generated site until the scheduled time passes.

## Settings

Scheduled publishing is controlled by two settings in the **Scheduling** category:

| Setting | Key | Default | Description |
|---------|-----|---------|-------------|
| Scheduled publish enabled | `ssg.scheduled.publish.enabled` | `true` | Turn scheduling on or off |
| Scheduled publish interval | `ssg.scheduled.publish.interval` | `15m` | How often to check for pending content |

### Changing settings

1. Go to your site in Clio
2. Open **Settings**
3. Find the **Scheduling** category
4. Edit the values

### Interval format

The interval uses Go duration format:

| Value | Meaning |
|-------|---------|
| `5m` | Every 5 minutes |
| `15m` | Every 15 minutes |
| `30m` | Every 30 minutes |
| `1h` | Every hour |
| `2h` | Every 2 hours |

The minimum interval is 1 minute. For production use, 15 minutes or more is recommended.

## Content Visibility Rules

| State | Visible on site? | Published by scheduler? |
|-------|-------------------|------------------------|
| Draft | No | No |
| Draft + future date | No | No |
| Draft + past date | No | No |
| Not draft + no date | Yes | No (no date to trigger) |
| Not draft + future date | No (until date passes) | Yes (when date arrives) |
| Not draft + past date | Yes | Yes (if after last publish) |

## Common Workflows

### Schedule a post for next Monday

1. Write your content
2. Uncheck **Draft**
3. Set **Published At** to next Monday at 9:00 AM
4. Save

The post won't appear on your site until Monday. On the first scheduler check after 9:00 AM, the site is regenerated and published with the new post included.

### Schedule a series of posts

1. Write all posts in the series
2. For each post, set a different **Published At** date (e.g., Monday, Wednesday, Friday)
3. Uncheck **Draft** on all of them
4. Save each one

Each post appears on schedule without any manual intervention.

### Disable scheduling temporarily

1. Go to **Settings** → **Scheduling**
2. Set **Scheduled publish enabled** to `false`

Pending content stays pending. When you re-enable, the next check picks up anything that's due.

## Manual Publish and Scheduling

Manual publish (clicking "Publish" in the site dashboard) works independently of the scheduler. When you publish manually:

- All publishable content is included (non-draft, published date not in the future)
- The site's last published timestamp is updated
- The scheduler uses this timestamp to avoid redundant publishes

You can use both manual and scheduled publishing together.

## Troubleshooting

### Scheduled post didn't appear

- Verify the content is **not a draft**
- Check that **Published At** is set and in the past
- Confirm **Scheduled publish enabled** is `true` in settings
- Check the Clio logs for scheduler messages

### Scheduler isn't running

Look for this log at startup:

```
Scheduler: started with interval 15m
```

If you see `Scheduler: no sites with scheduling enabled`, enable it in settings and restart Clio.

### Publish failed

The scheduler logs errors when publish fails:

```
Scheduler: publish failed for site my-site: ...
```

Common causes:
- Git repository not configured (set `ssg.publish.repo.url` in settings)
- Invalid authentication token
- Network issues

The scheduler retries on the next interval automatically.
