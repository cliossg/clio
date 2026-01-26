-- +migrate Up
CREATE TABLE IF NOT EXISTS section_images (
    id TEXT PRIMARY KEY,
    section_id TEXT NOT NULL,
    image_id TEXT NOT NULL,
    is_header INTEGER DEFAULT 0,
    is_featured INTEGER DEFAULT 0,
    order_num INTEGER DEFAULT 0,
    created_at TIMESTAMP,
    FOREIGN KEY (section_id) REFERENCES section(id) ON DELETE CASCADE,
    FOREIGN KEY (image_id) REFERENCES image(id) ON DELETE CASCADE,
    UNIQUE(section_id, image_id)
);

CREATE INDEX IF NOT EXISTS idx_section_images_section_id ON section_images(section_id);
CREATE INDEX IF NOT EXISTS idx_section_images_image_id ON section_images(image_id);

-- +migrate Down
DROP TABLE IF EXISTS section_images;
