# Google Search

Google Search integration adds site search to your generated site using Google Programmable Search Engine (CSE). Visitors can search your content directly from the navigation bar.

## Quick Start

1. Create a search engine at [Google Programmable Search Engine](https://programmablesearchengine.google.com/)
2. Go to **Settings** → **Search** category
3. Set **Google Search enabled** to `true`
4. Enter your **Google Search ID**
5. Generate and publish your site

A search icon appears in the navigation bar, and a dedicated search results page is generated at `/search/`.

## Settings

| Setting | Key | Default | Description |
|---------|-----|---------|-------------|
| Google Search enabled | `ssg.search.google.enabled` | `true` | Turn site search on or off |
| Google Search ID | `ssg.search.google.id` | (empty) | Your Google CSE ID |

Both the flag and the ID must be set for search to appear. If the ID is empty, no search UI is included.

## How It Works

When search is enabled, Clio adds two things to your site:

### Search Icon in Navigation

A magnifying glass icon appears at the right side of the navigation bar. Clicking it expands a search box with a smooth animation. The search box uses Google's CSE searchbox widget and directs results to the `/search/` page.

### Search Results Page

Clio generates a `/search/index.html` page that displays search results using Google's CSE results widget. This page uses the same layout, navigation, and footer as the rest of your site.

## Getting Your Search ID

1. Go to [Programmable Search Engine](https://programmablesearchengine.google.com/)
2. Click **Add** to create a new search engine
3. Enter your site's domain (e.g. `example.com`)
4. Give it a name
5. Click **Create**
6. Copy the **Search engine ID** from the overview page
7. Paste it in Clio's settings

The search engine is free for public websites.

## Search Indexing

Google CSE searches Google's index of your site. For results to appear:

- Your site must be publicly accessible
- Google must have crawled your pages (this can take days for new sites)
- Having a `sitemap.xml` helps (Clio generates one automatically when a base URL is configured)

You can check indexing status in [Google Search Console](https://search.google.com/search-console/).

## Troubleshooting

### Search icon not showing

- Verify **Google Search enabled** is `true`
- Check that **Google Search ID** is not empty
- Regenerate the site after changing settings

### No search results

- Your site may not be indexed by Google yet
- Verify the CSE is configured for the correct domain
- Check that your site is publicly accessible (CSE doesn't work on localhost)

### Search box doesn't expand

- Check the browser console for JavaScript errors
- Verify the CSE script loaded (ad blockers may block it)
