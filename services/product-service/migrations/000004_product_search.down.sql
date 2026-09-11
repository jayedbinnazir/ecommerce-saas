DROP INDEX IF EXISTS products_search_idx;
ALTER TABLE products DROP COLUMN IF EXISTS search;
