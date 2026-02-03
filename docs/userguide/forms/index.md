# Contact Forms

Contact forms let visitors send you messages directly from your generated site. Submissions are saved to the database and appear in the Messages inbox in the dashboard.

## Why use Contact Forms?

- **No third-party service**: Messages go straight to your Clio instance
- **Simple syntax**: Add a form with a single Markdown code block
- **Spam protection**: Built-in honeypot field and rate limiting
- **Dashboard inbox**: Read, manage, and delete messages from the admin UI
- **Works with Cloudflare Tunnel**: No extra configuration when serving your site via tunnel

## Quick Start

1. Go to **Settings** → **Forms** category
2. Set **Forms enabled** to `true`
3. Add a form to any content:

````markdown
```form
type: contact
```
````

4. Generate your site
5. Submissions appear under **Messages** in the dashboard

That's it if you serve your site via Cloudflare Tunnel (see [Deployment Scenarios](#deployment-scenarios) below). If you publish to GitHub Pages, there's one extra setting to configure.

## Settings

| Setting | Key | Default | Description |
|---------|-----|---------|-------------|
| Forms enabled | `ssg.forms.enabled` | `false` | Turn form submissions on or off |
| Forms endpoint URL | `ssg.forms.endpoint_url` | (empty) | Override the form action URL (only needed for GitHub Pages) |
| Forms allowed origins | `ssg.forms.allowed_origins` | (empty) | Comma-separated list of allowed origins for CORS |
| Forms rate limit | `ssg.forms.rate_limit` | `5` | Maximum submissions per IP address per hour |

## Syntax

Forms use a fenced code block with the `form` language, following the same pattern as [Embeds](../embeds/index.md):

````markdown
```form
type: contact
```
````

### Fields

| Field  | Description                   | Required |
|--------|-------------------------------|----------|
| `type` | The form type. Currently only `contact` is supported | Yes |

## How It Works

### Form generation

When your site is generated, Clio replaces the code block with an HTML form:

```html
<form class="clio-form" action="/api/v1/forms/submit" method="POST">
  <input type="hidden" name="_site" value="your-site-uuid">
  <input type="hidden" name="_form" value="contact">
  <input type="text" name="_honeypot" style="display:none" tabindex="-1" autocomplete="off">
  <div class="form-field">
    <label for="cf-name">Name</label>
    <input type="text" id="cf-name" name="name" required>
  </div>
  <div class="form-field">
    <label for="cf-email">Email</label>
    <input type="email" id="cf-email" name="email" required>
  </div>
  <div class="form-field">
    <label for="cf-message">Message</label>
    <textarea id="cf-message" name="message" rows="5" required></textarea>
  </div>
  <button type="submit">Send</button>
</form>
```

By default the form action is a relative path (`/api/v1/forms/submit`). This works when the site is served from Clio itself (via tunnel). If **Forms endpoint URL** is configured, the form uses that absolute URL instead.

### Submission flow

1. A visitor fills out the form on your published site
2. The browser sends a POST request to the forms endpoint
3. The server validates the submission (honeypot, required fields, rate limit)
4. The message is saved to the database
5. You see it in the Messages inbox

### Spam protection

Two mechanisms protect against spam:

- **Honeypot field**: A hidden input that real users never fill in. If a bot fills it, the server silently accepts the request (returns 200) but does not save the message.
- **Rate limiting**: Each IP address is limited to a configurable number of submissions per hour (default: 5). Requests beyond the limit receive a `429 Too Many Requests` response.

## Deployment Scenarios

Contact forms require a running Clio instance to receive submissions. How you set this up depends on how you deploy your site.

### Scenario 1: Cloudflare Tunnel (recommended)

In this scenario, your site is served directly from Clio through a Cloudflare Tunnel. There's no GitHub Pages involved. This is the simplest setup because the site and the forms endpoint live on the same server.

**How it works:**

```
Visitor → Cloudflare Tunnel → Clio preview server (:3000)
                                ├── /           → static site
                                └── /api/v1/forms/submit → forms handler
```

**Setup:**

1. Run Clio with Docker and a tunnel (see [Self-Hosting with Docker](../docker/index.md)):

```bash
make docker-up-quick-tunnel
```

2. Enable forms in **Settings** → set **Forms enabled** to `true`
3. Add a `form` block to your content and generate the site
4. Done. The form posts to `/api/v1/forms/submit` on the same server

No **endpoint URL** configuration is needed. The form uses a relative path that resolves to the same server serving the site.

**With a permanent tunnel** (custom domain like `myblog.com`):

```bash
# Set your tunnel token in .env
CLOUDFLARE_TUNNEL_TOKEN=your-token-here

# Start with the named tunnel
make docker-up-tunnel
```

Configure your tunnel in the Cloudflare dashboard to route your domain to `http://app:3000`. Your site and forms both work through the same domain.

### Scenario 2: GitHub Pages + Cloudflare Tunnel

In this scenario, your static site is published to GitHub Pages, but Clio runs on your machine (or a server) with a Cloudflare Tunnel exposing only the forms endpoint. Visitors see the site on GitHub Pages but form submissions go to your Clio instance.

**How it works:**

```
Visitor → GitHub Pages (myblog.github.io)
            └── form action="https://forms.myblog.com/api/v1/forms/submit"
                    └── Cloudflare Tunnel → Clio preview server (:3000)
```

**Setup:**

1. Run Clio with a named Cloudflare Tunnel on your machine:

```bash
make docker-up-tunnel
```

2. Configure the tunnel in the Cloudflare dashboard to route a subdomain (e.g. `forms.myblog.com`) to `http://app:3000`
3. In Clio **Settings**:
   - Set **Forms enabled** to `true`
   - Set **Forms endpoint URL** to `https://forms.myblog.com`
   - Set **Forms allowed origins** to `https://myblog.github.io` (your GitHub Pages domain)
4. Add a form block to your content
5. Generate and publish to GitHub Pages as usual

The generated form will use the absolute URL `https://forms.myblog.com/api/v1/forms/submit` as its action.

**Important**: The **allowed origins** setting is required in this scenario because the form is submitted cross-origin (GitHub Pages domain → your tunnel domain). Without it, browsers will block the request.

### Scenario comparison

| | Tunnel only | GitHub Pages + Tunnel |
|---|---|---|
| Site served from | Clio (via tunnel) | GitHub Pages |
| Forms endpoint URL needed | No (relative path) | Yes (absolute URL) |
| Allowed origins needed | No | Yes |
| Clio must be running | Always | Only to receive submissions |
| Custom domain | Via Cloudflare Tunnel | Via GitHub Pages + tunnel subdomain |
| Complexity | Low | Medium |

### Which scenario should I choose?

**Use Tunnel only** if you want the simplest setup and don't mind keeping Clio running. Everything works from one URL with zero configuration beyond enabling forms.

**Use GitHub Pages + Tunnel** if you want the reliability of GitHub Pages for serving static content but still want a contact form. Your site stays up even if Clio goes offline — visitors just can't submit the form during downtime.

## Dashboard: Messages

The Messages section in the dashboard shows all form submissions for the current site.

### Message list

Navigate to **Messages** in the sidebar. The list shows:

- Sender name and email
- Message preview
- Date
- Status badge: **new** (unread) or **read**

An unread count badge appears in the header when there are new messages.

### Message detail

Click a message to view the full content. The message is automatically marked as read when you open it.

From the detail view you can:

- **Mark as Read** (if not already read)
- **Delete** the message (with confirmation)

## Common Workflows

### Adding a contact page

Create a new page in your site and add the form block:

````markdown
# Contact

Have a question or want to get in touch? Fill out the form below.

```form
type: contact
```

We'll get back to you as soon as possible.
````

### Using forms in articles

You can include a form anywhere in your content, not just on dedicated pages:

````markdown
## Feedback

Did you find this article helpful? Let us know:

```form
type: contact
```
````

## Troubleshooting

### Form doesn't appear in generated site

- Verify **Forms enabled** is `true` in Settings
- Check that the code block uses exactly `form` as the language
- The `type` field must be `contact` (other types are not yet supported)
- Regenerate the site after changing settings

### Submissions not being saved

- Check that Clio is running and reachable via tunnel
- Verify the site ID in the form's hidden field matches an existing site
- Check the Clio logs for errors (`make docker-logs`)

### Getting 429 Too Many Requests

- The rate limiter restricts submissions per IP per hour
- Wait for the window to reset, or increase the **Forms rate limit** setting
- The default limit is 5 submissions per hour per IP

### CORS errors in browser console

This only applies to [Scenario 2](#scenario-2-github-pages--cloudflare-tunnel) (GitHub Pages + Tunnel):

- Set **Forms allowed origins** to your GitHub Pages domain (e.g. `https://myblog.github.io`)
- Include the protocol (`https://`) in the origin
- Multiple origins can be separated by commas

### Form submits but nothing appears in Messages

- Check if the honeypot field was filled (bots trigger this). The server returns 200 but does not save.
- Verify you're looking at the correct site's messages
