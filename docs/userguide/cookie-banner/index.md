# Cookie Banner

The cookie consent banner informs visitors that your site uses cookies. It can be enabled independently or appears automatically when Google Analytics is active.

## Quick Start

1. Go to **Settings** → **Site** category
2. Set **Cookie banner enabled** to `true`
3. Optionally customize the **Cookie banner text**
4. Generate and publish your site

## Settings

| Setting | Key | Default | Description |
|---------|-----|---------|-------------|
| Cookie banner enabled | `ssg.cookie.banner.enabled` | `true` | Show the cookie consent banner |
| Cookie banner text | `ssg.cookie.banner.text` | (see below) | Custom consent message |

Default text when the setting is empty:

> This site uses cookies to improve your experience. By continuing to use this site, you accept our use of cookies.

## When Does It Appear?

The banner shows when **either** condition is true:

| Condition | Banner visible? |
|-----------|----------------|
| Cookie banner enabled = `true` | Yes |
| Analytics enabled with a valid ID | Yes |
| Both disabled | No |

This means you don't need to manually enable the cookie banner when using Google Analytics — it activates automatically.

## How It Works

1. On first visit, the banner appears at the bottom of the page
2. The visitor clicks **Accept**
3. The preference is saved in `localStorage` (`cookiesAccepted = true`)
4. The banner does not appear again for that browser

The banner uses no external dependencies. Acceptance is stored client-side only.

## Customizing the Text

1. Go to **Settings** → **Site** category
2. Edit **Cookie banner text** with your message
3. Leave empty to use the default text

The text supports plain text only (no HTML).

## Styling

The banner is styled with the `.cookie-banner` class in the default theme:

- Fixed to the bottom of the viewport
- Dark background (`#1f2937`) with white text
- Accept button in blue (`#2563eb`)
- Responsive: stacks vertically on mobile

Custom layouts can override these styles.

## Troubleshooting

### Banner doesn't appear

- Check that **Cookie banner enabled** is `true` or that analytics is active
- Clear your browser's `localStorage` (the banner won't show if you already accepted)
- Regenerate the site after changing settings

### Banner keeps reappearing

- Check that JavaScript is enabled in the browser
- Verify `localStorage` is not being cleared by browser privacy settings
