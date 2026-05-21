-- ===================================================================
-- Migration : 001_create_hookzen_users_table.down.sql
-- ===================================================================

-- 1. Drop trigger dependencies to clear relation attachments
DROP TRIGGER IF EXISTS trg_user_preferences_updated_at ON user_preferences;
DROP TRIGGER IF EXISTS trg_2fa_updated_at ON two_factor_auth;
DROP TRIGGER IF EXISTS trg_hookzen_users_updated_at ON hookzen_users;

-- 2. Drop dependent constraints and tables in order of references
DROP TABLE IF EXISTS user_preferences;
DROP TABLE IF EXISTS two_factor_auth;
DROP TABLE IF EXISTS hookzen_users;

-- 3. Clear data types allocated for application runtime structures
DROP TYPE IF EXISTS two_factor_method;