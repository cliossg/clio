# Images

Images collects all images uploaded across every content item in the current site. Click the **Images** card on the [site dashboard](../sites/dashboard/index.md) to open the image gallery.

## The Image Gallery

The gallery shows all images as a grid of thumbnails. Each thumbnail displays the filename and title below it. Click any image to view its details.

Images are uploaded from within the [content editor](../content/index.md) (either as header images or content images). The gallery provides a centralized place to review and edit the metadata for all of them.

## Image Details

Click an image to see its detail page. This shows:

- The full-size image
- **File Name**: the original filename
- **File Path**: the internal path (includes a unique ID to avoid collisions)
- **Title**: a short descriptive title
- **Alt Text**: the description used for screen readers and SEO
- **Created** and **Updated**: timestamps

From here you can click **Edit Details** to update the metadata, or **Delete** to remove the image.

---

## Editing Image Details

Click **Edit Details** on the image detail page. The edit form lets you change the following fields:

| Field | Description |
|---|---|
| **Title** | A short title for the image |
| **Alt Text** | Describe the image for screen readers and SEO |
| **Attribution** | Credit the image author or source (e.g. "Photo by John Doe") |
| **Attribution URL** | Link to the author's page or the original source |

Click **Save** to apply changes or **Cancel** to discard.

These values affect how the image is rendered on the generated site. The alt text is used in the HTML `alt` attribute. The attribution and URL are displayed as image credits in the default templates.

The image file itself cannot be replaced. Changing the file would alter the filename and break references in content that uses it. If you need a different image, upload a new one and update your content to reference it instead.

---

## Uploading Images

Images are not uploaded from the gallery. They are uploaded from within a content item:

- **Header image**: uploaded from the header image area at the top of the content editor
- **Content images**: uploaded from the "Content Images" section below the editor

Once uploaded, images appear in the gallery automatically. See the [Content](../content/index.md) guide for details on uploading.

---

## Deleting Images

Click **Delete** on the image detail page. This removes the image from the database and from disk. Any content that references the deleted image will show a broken image after the site is regenerated.
