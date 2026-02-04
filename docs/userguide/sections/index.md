# Sections

Sections organize your content into groups like "Blog", "Docs", or "Tutorials". Click the **Sections** card on the [site dashboard](../sites/dashboard/index.md) to open the sections list.

## The Sections List

The list shows all sections for the current site. Each row displays:

| Column | Description |
|---|---|
| **Name** | The section name (clickable) |
| **Path** | The URL path for this section |
| **Layout** | The layout assigned to this section, or empty if using the site default |
| **Actions** | Edit button |

---

## Creating a Section

Click **New Section** in the top-right corner. The form has the following fields:

| Field | Description |
|---|---|
| **Name** | The display name for the section (e.g. "Blog") |
| **Description** | A short description of the section's purpose |
| **URL Path** | The path used in the generated site's URL structure (e.g. `/blog`) |
| **Layout** | Optional. Choose a layout to override the site's default layout for content in this section. If left empty, the site default is used. |
| **Hero Title Style** | Light or Dark. Controls the text contrast on the section's hero area, similar to the [header image toggle](../content/index.md) on content items. |

Click **Create** to save the section.

---

## Editing a Section

Click **Edit** next to a section in the list. The edit form includes the same fields as the create form, plus:

- **Section Header Image**: an image displayed on the section's index page when the site is generated. Includes its own Light/Dark toggle for text contrast.
- **Section Images**: additional images associated with the section.

Click **Update** to save changes or **Cancel** to discard.

---

## Layouts and Sections

Each section can optionally use a different layout from the site default. This lets you give different areas of your site a distinct look. For example, your blog section might use a layout with a sidebar, while your documentation section uses a full-width layout.

If no layout is selected for a section, content in that section uses the site's default layout.

---

## Deleting a Section

Click **Delete** next to a section in the list. Removing a section does not delete the content assigned to it. Those content items become unassigned and appear with "None" in the Section column of the [content list](../content/index.md).
