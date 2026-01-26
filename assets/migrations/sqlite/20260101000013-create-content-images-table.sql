-- +migrate Up
CREATE TABLE IF NOT EXISTS content_images (
    id TEXT PRIMARY KEY,
    content_id TEXT NOT NULL,
    image_id TEXT NOT NULL,
    is_header INTEGER DEFAULT 0,
    is_featured INTEGER DEFAULT 0,
    order_num INTEGER DEFAULT 0,
    created_at TIMESTAMP,
    FOREIGN KEY (content_id) REFERENCES content(id) ON DELETE CASCADE,
    FOREIGN KEY (image_id) REFERENCES image(id) ON DELETE CASCADE,
    UNIQUE(content_id, image_id)
);

CREATE INDEX IF NOT EXISTS idx_content_images_content_id ON content_images(content_id);
CREATE INDEX IF NOT EXISTS idx_content_images_image_id ON content_images(image_id);

-- +migrate Down
DROP TABLE IF EXISTS content_images;
