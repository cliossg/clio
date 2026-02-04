# REST API

Clio provides a local REST API for managing content programmatically. Use it from editors like Neovim, scripts, or any HTTP client.

## Why use the REST API?

- **Write from your editor**: Create and edit posts directly from Neovim, VSCode, or any editor with HTTP support
- **Automate workflows**: Script publishing, backup, content generation, and batch operations
- **Build integrations**: Custom dashboards, CI/CD pipelines, webhook handlers
- **Headless CMS**: Use Clio as a content backend for other applications or static site generators

## Use Cases

### Neovim

Imagine writing a post in Markdown, hitting `<leader>cp`, and having it live on your site. The API makes this possible with a few lines of Lua. Here's a starting point:

```lua
-- ~/.config/nvim/lua/clio.lua
-- A minimal example — a proper clio.nvim plugin could add
-- Telescope pickers for sites/sections, draft toggling,
-- live preview, and :ClioPublish commands. PRs welcome ;)

local token = "clio_xxxxxxxxxxxxxxxxxxxx"  -- from /api/tokens
local site = "your-site-uuid"              -- from GET /api/v1/sites
local section = "your-section-uuid"        -- from GET /api/v1/sites/:id

local function publish_to_clio()
  local lines = vim.api.nvim_buf_get_lines(0, 0, -1, false)
  local body = table.concat(lines, "\n")
  local cmd = string.format(
    'curl -s -X POST -H "Authorization: Bearer %s" '
    .. '-H "Content-Type: application/json" '
    .. '-d \'{"heading":"%s","body":"%s","section_id":"%s"}\' '
    .. 'http://localhost:8080/api/v1/sites/%s/posts',
    token, vim.fn.expand("%:t:r"), body:gsub('"', '\\"'), section, site
  )
  vim.fn.system(cmd)
  vim.notify("Published to Clio!", vim.log.levels.INFO)
end

vim.keymap.set("n", "<leader>cp", publish_to_clio, { desc = "Clio: publish buffer" })
```

A full `clio.nvim` plugin could offer Telescope integration to browse and pick sites and sections, `:ClioPublish` and `:ClioDraft` commands, status line indicators, and even live preview. The REST API provides everything you need, all the endpoints are there.

### VSCode

Create a task or extension that syncs Markdown files from a workspace folder to Clio on save:

```json
// .vscode/tasks.json
{
  "label": "Publish to Clio",
  "type": "shell",
  "command": "curl -s -X POST -H 'Authorization: Bearer clio_xxxxxxxxxxxxxxxxxxxx' -H 'Content-Type: application/json' -d @${file} http://localhost:8080/api/v1/sites/your-site-uuid/posts"
}
```

### Shell Scripts

Batch operations, cron jobs, and automation pipelines:

```bash
#!/bin/bash
# publish-all.sh — publish all .md files in a directory
TOKEN="clio_xxxxxxxxxxxxxxxxxxxx"        # from /api/tokens
SITE="your-site-uuid"                    # from GET /api/v1/sites
SECTION="your-section-uuid"              # from GET /api/v1/sites/:id

for file in content/*.md; do
  heading=$(head -1 "$file" | sed 's/^# //')
  body=$(tail -n +3 "$file")
  curl -s -X POST \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"heading\":\"$heading\",\"body\":\"$body\",\"section_id\":\"$SECTION\"}" \
    "http://localhost:8080/api/v1/sites/$SITE/posts"
done
```

## Getting Started

### 1. Create a Token

Navigate to **API** in the dashboard navigation, or visit `/api/tokens`. Click **New Token**, give it a name, and copy the generated token. The token is shown only once.

### 2. Make Requests

```bash
export CLIO_TOKEN="your-token-here"

curl -H "Authorization: Bearer $CLIO_TOKEN" \
  http://localhost:8080/api/v1/sites
```

## Authentication

All API requests require a Bearer token in the `Authorization` header:

```
Authorization: Bearer <token>
```

Invalid or expired tokens return `401 Unauthorized`.

## Endpoints

### Tokens

| Method | Path                 | Description        |
| ------ | -------------------- | ------------------ |
| POST   | `/api/v1/tokens`     | Create a new token |
| GET    | `/api/v1/tokens`     | List your tokens   |
| DELETE | `/api/v1/tokens/:id` | Revoke a token     |

### Sites

| Method | Path                | Description      |
| ------ | ------------------- | ---------------- |
| GET    | `/api/v1/sites`     | List all sites   |
| GET    | `/api/v1/sites/:id` | Get site details |

### Posts

| Method | Path                               | Description    |
| ------ | ---------------------------------- | -------------- |
| GET    | `/api/v1/sites/:id/posts`          | List all posts |
| GET    | `/api/v1/sites/:id/posts/:post_id` | Get a post     |
| POST   | `/api/v1/sites/:id/posts`          | Create a post  |
| PUT    | `/api/v1/sites/:id/posts/:post_id` | Update a post  |
| DELETE | `/api/v1/sites/:id/posts/:post_id` | Delete a post  |

### Publishing

| Method | Path                         | Description               |
| ------ | ---------------------------- | ------------------------- |
| POST   | `/api/v1/sites/:id/generate` | Generate HTML             |
| POST   | `/api/v1/sites/:id/publish`  | Generate + publish to git |
| POST   | `/api/v1/sites/:id/backup`   | Backup markdown to git    |

## Examples

### List sites

```bash
curl -s -H "Authorization: Bearer $CLIO_TOKEN" \
  http://localhost:8080/api/v1/sites | jq
```

### Create a post

```bash
curl -s -X POST \
  -H "Authorization: Bearer $CLIO_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "section_id": "SECTION-UUID",
    "heading": "My New Post",
    "body": "# Hello\n\nThis is my post content.",
    "summary": "A short summary",
    "draft": false
  }' \
  http://localhost:8080/api/v1/sites/SITE-UUID/posts | jq
```

### Update a post

```bash
curl -s -X PUT \
  -H "Authorization: Bearer $CLIO_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"body": "Updated content here."}' \
  http://localhost:8080/api/v1/sites/SITE-UUID/posts/POST-UUID | jq
```

### Publish

```bash
curl -s -X POST \
  -H "Authorization: Bearer $CLIO_TOKEN" \
  http://localhost:8080/api/v1/sites/SITE-UUID/publish | jq
```

## Error Responses

Errors return JSON with a structured error object:

```json
{
  "error": {
    "code": "not_found",
    "message": "Post not found"
  }
}
```

Common error codes: `unauthorized`, `invalid_id`, `not_found`, `validation_error`, `internal_error`, `config_error`.

## Token Management

- Tokens are scoped to the user who created them
- Token values are hashed before storage (SHA-256); the raw token is shown only at creation
- Revoke tokens from the dashboard at `/api/tokens`
- Each request with a valid token updates the "Last Used" timestamp

## Security

- The API is only accessible from localhost (same as the rest of Clio)
- Tokens use cryptographically random 32-byte values
- Only the SHA-256 hash of each token is stored in the database
