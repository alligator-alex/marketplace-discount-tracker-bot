CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP(0) DEFAULT NOW(),
    updated_at TIMESTAMP(0) DEFAULT NOW(),
    scraped_at TIMESTAMP(0),
    slug VARCHAR NOT NULL UNIQUE,
    telegram_chat_id BIGINT NOT NULL,
    telegram_user_id BIGINT NOT NULL,
    url VARCHAR NOT NULL,
    marketplace SMALLINT NOT NULL,
    title VARCHAR NOT NULL,
    threshold_price INTEGER NOT NULL,
    current_price INTEGER,
    out_of_stock BOOLEAN DEFAULT FALSE
);

CREATE UNIQUE INDEX idx_chat_user_product ON products (telegram_chat_id, telegram_user_id, url);