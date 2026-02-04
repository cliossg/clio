# Contributors

Contributors are the people who create content for a site. Click **Contributors** in the navigation bar to manage contributors for the current site.

## The Contributors List

The list shows all contributors for the current site. Each row displays:

| Column | Description |
|---|---|
| **Handle** | The contributor's handle (e.g. `@johndoe`) |
| **Name** | Full name |
| **Actions** | Edit and Profile buttons |

---

## Creating a Contributor

Click **New Contributor** in the top-right corner. The form has the following fields:

| Field | Description |
|---|---|
| **Handle** | Internal identifier for this contributor (e.g. `johndoe`) |
| **Name** | First name |
| **Surname** | Last name |
| **Bio** | A short biography |

Click **Save** to create the contributor.

---

## Editing a Contributor

Click **Edit** next to a contributor in the list. The edit form shows the same fields as the create form, plus an **Edit Profile** button in the top-right corner that opens the profile editor.

---

## The Profile

Each contributor can have a public profile. The profile controls what appears on the contributor's author page on the generated site (rendered at `/authors/{handle}/`). Click **Edit Profile** from the contributor edit page, or **Profile** from the contributors list.

The profile form has the following fields:

| Field | Description |
|---|---|
| **Profile Photo** | A photo displayed on the author page. Upload, change, or remove. |
| **Slug** | URL-friendly identifier for the author page |
| **Display Name** | Public first name |
| **Display Surname** | Public last name |
| **Bio** | Public biography |

### Social Links

Below the bio, the profile includes fields for social media URLs. Enter the full URL for each platform.

The main platforms shown by default: YouTube, Instagram, X, TikTok, LinkedIn, GitHub, WhatsApp, Telegram, Reddit.

A **More platforms** section expands to reveal additional options: Messenger, Snapchat, Pinterest, Tumblr, Discord, Twitch, Signal, Viber, LINE, KakaoTalk, WeChat, QQ, Douyin, Kuaishou, Weibo, and others.

Only platforms with a URL filled in are displayed on the generated author page.

Click **Save** to apply changes or **Cancel** to discard.

---

## Contributor vs Profile

The contributor and its profile currently share some fields (name, surname, bio). This duplication is unnecessary. A future version of Clio will remove the repeated fields so they only appear in one place.

---

## Assigning Contributors to Content

By default, content is attributed to the logged-in user who creates it. You can optionally select a contributor from the **Contributor** dropdown in the [content editor](../content/index.md) (inside the "Section, Kind, Contributor and Summary" panel). When a contributor is selected, their profile replaces the default author on the generated page.

This is useful for guest posts, collaborations, or cross-postings between sites. Clio is primarily a self-hosted, single-user application, but it is common for blogs and sites to feature contributions from other people. Contributors make this straightforward without requiring additional user accounts, since Clio is a static site generator and not a platform where people register.

Each contributor belongs to a single site.

---

## Deleting a Contributor

Click **Delete** next to a contributor in the list. Removing a contributor does not delete the content assigned to them. Those content items lose their author attribution.
