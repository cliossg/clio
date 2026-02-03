# Google Analytics

Google Analytics integration lets you track visitor traffic on your generated site using Google's measurement platform.

## Quick Start

1. Go to **Settings** → **Analytics** category
2. Set **Google Analytics enabled** to `true`
3. Enter your **Google Analytics ID** (e.g. `G-XXXXXXXXXX`)
4. Generate and publish your site

The tracking snippet loads automatically on every page.

## Settings

| Setting | Key | Default | Description |
|---------|-----|---------|-------------|
| Google Analytics enabled | `ssg.analytics.enabled` | `true` | Turn analytics tracking on or off |
| Google Analytics ID | `ssg.analytics.id` | (empty) | Your Google Analytics measurement ID |

Both the flag and the ID must be set for analytics to appear. If the ID is empty, no tracking code is included regardless of the flag.

## How It Works

When enabled, Clio adds the Google gtag.js snippet to the `<head>` of every generated page. The snippet is placed at the very top of `<head>` as Google recommends, so it loads before other resources.

The generated code looks like:

```html
<script async src="https://www.googletagmanager.com/gtag/js?id=G-XXXXXXXXXX"></script>
<script>
    window.dataLayer = window.dataLayer || [];
    function gtag(){dataLayer.push(arguments);}
    gtag('js', new Date());
    gtag('config', 'G-XXXXXXXXXX');
</script>
```

## Getting Your Analytics ID

1. Go to [Google Analytics](https://analytics.google.com/)
2. Create a property for your site (or use an existing one)
3. Go to **Admin** → **Data Streams** → **Web**
4. Copy the **Measurement ID** (starts with `G-`)
5. Paste it in Clio's settings

## Cookie Banner

When analytics is active, the cookie consent banner appears automatically even if the cookie banner setting is disabled. This ensures compliance with cookie consent requirements. See the [Cookie Banner](../cookie-banner/index.md) guide for details.

## Troubleshooting

### Analytics not appearing on site

- Verify **Google Analytics enabled** is `true`
- Check that **Google Analytics ID** is not empty
- Regenerate and republish the site after changing settings
- Check if a browser ad blocker is hiding the snippet (view page source to confirm it's there)

### Data not showing in Google Analytics

- It can take 24-48 hours for data to appear in a new property
- Verify the measurement ID matches your Google Analytics property
- Check that your site's domain matches the data stream configuration
