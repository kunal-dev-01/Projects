-- ===================================================================
-- Migration  : 000_enable_extentions.down.sql
-- ===================================================================

DROP EXTENSION IF EXISTS "citext";
DROP FUNCTION IF EXISTS fn_set_updated_at();