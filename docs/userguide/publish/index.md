# Publish

Publish deploys your generated site to a Git repository. Click the **Publish** card on the [site dashboard](../sites/dashboard/index.md) to deploy.

## How It Works

Clicking Publish generates the static site (the same process as [Preview](../preview/index.md)) and pushes the result to the configured Git repository and branch. This is how you deploy your site to GitHub Pages.

If the publish repository is not configured, Clio shows an error message and redirects you back to the dashboard.

## Configuration

Before you can publish, you need to configure the repository settings in the [Settings](../sites/dashboard/index.md#settings) page:

| Setting | Description | Default |
|---|---|---|
| **Publish repository URL** | Git repository URL (HTTPS or SSH) | (required) |
| **Publish branch** | The branch to push generated files to | `gh-pages` |
| **Publish auth token** | Authentication token for HTTPS repositories (not needed for SSH) | (required for HTTPS) |
| **Commit user name** | Git author name used in commits | `Clio Bot` |
| **Commit user email** | Git author email used in commits | `clio@localhost` |

The commit user name and email are shared with the [Backup](../backup/index.md) feature.

### HTTPS vs SSH

For HTTPS repository URLs, an auth token is required. This is typically a personal access token from GitHub.

For SSH repository URLs (e.g. `git@github.com:user/repo.git`), no auth token is needed. Clio uses the SSH keys available on the server.

## Scheduled Publishing

Clio can publish automatically on a schedule. When enabled, it checks for content whose publish date has passed and regenerates the site at regular intervals. This is useful for publishing content at a future date without manual intervention.

| Setting | Description | Default |
|---|---|---|
| **Scheduled publish enabled** | Enable automatic publishing of scheduled content | `true` |
| **Scheduled publish interval** | How often to check for scheduled content (e.g. `1h`, `30m`, `15m`) | `15m` |

For details on setting publish dates on content, see the [Scheduled Publishing](../scheduling/index.md) guide.
