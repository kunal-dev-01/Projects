-- ===================================================================
-- Migration : 001_create_hookzen_users_table.up.sql 
-- ===================================================================

-- ───────────────────────────────────────────────────────────────────
-- 1. ENUMS
-- ───────────────────────────────────────────────────────────────────

CREATE TYPE two_factor_method AS ENUM (
    'totp',
    'backup_code'
);

-- ───────────────────────────────────────────────────────────────────
-- 2. TABLE: hookzen_users
-- ───────────────────────────────────────────────────────────────────

CREATE TABLE hookzen_users (
    id                           UUID         NOT NULL DEFAULT gen_random_uuid(),

    -- Contact Information (Using CITEXT from 000 migration)
    email                        CITEXT       NOT NULL,
    email_verified               BOOLEAN      NOT NULL DEFAULT FALSE,
    email_verified_at            TIMESTAMPTZ,

    -- Authentication Core
    password_hash                TEXT         NOT NULL,
    password_changed_at          TIMESTAMPTZ,
    password_reset_requested_at  TIMESTAMPTZ,

    -- Profile Elements
    full_name                    VARCHAR(200) NOT NULL DEFAULT '',

    -- Localization
    timezone                     VARCHAR(64)  NOT NULL DEFAULT 'UTC',
    locale                       VARCHAR(10)  NOT NULL DEFAULT 'en',

    -- Account State Management
    is_active                    BOOLEAN      NOT NULL DEFAULT TRUE,

    -- Security Auditing Fields
    last_login_at                TIMESTAMPTZ,
    last_login_ip                INET,

    -- Two-Factor Flags (Denormalized for hot auth-path performance)
    two_factor_enabled           BOOLEAN      NOT NULL DEFAULT FALSE,
    two_factor_enabled_at        TIMESTAMPTZ,

    -- Lifecycle Monitoring Timestamps
    created_at                   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at                   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at                   TIMESTAMPTZ,

    -- Constraints & Integrity Controls
    CONSTRAINT hookzen_users_pkey
        PRIMARY KEY (id),
    CONSTRAINT hookzen_users_email_unique
        UNIQUE (email),
    CONSTRAINT chk_email_verified_at
        CHECK (email_verified = FALSE OR email_verified_at IS NOT NULL),
    CONSTRAINT chk_2fa_enabled_at
        CHECK (two_factor_enabled = FALSE OR two_factor_enabled_at IS NOT NULL)
);

-- ───────────────────────────────────────────────────────────────────
-- 3. HIGH-PERFORMANCE PARTIAL INDEXES
-- ───────────────────────────────────────────────────────────────────

CREATE INDEX idx_hookzen_users_email
    ON hookzen_users (email)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_hookzen_users_active
    ON hookzen_users (id)
    WHERE is_active = TRUE
      AND deleted_at IS NULL;

CREATE INDEX idx_hookzen_users_deleted_at
    ON hookzen_users (deleted_at)
    WHERE deleted_at IS NOT NULL;

CREATE INDEX idx_hookzen_users_last_login_at
    ON hookzen_users (last_login_at DESC NULLS LAST)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_hookzen_users_2fa_enabled
    ON hookzen_users (id)
    WHERE two_factor_enabled = TRUE
      AND deleted_at IS NULL;

-- ───────────────────────────────────────────────────────────────────
-- 4. TABLE: two_factor_auth
-- ───────────────────────────────────────────────────────────────────

CREATE TABLE two_factor_auth (
    id                        UUID              NOT NULL DEFAULT gen_random_uuid(),
    user_id                   UUID              NOT NULL,

    -- TOTP Parameters
    totp_secret               TEXT              NOT NULL,
    totp_algorithm            VARCHAR(10)       NOT NULL DEFAULT 'SHA1',
    totp_digits               SMALLINT          NOT NULL DEFAULT 6,
    totp_period               SMALLINT          NOT NULL DEFAULT 30,

    -- Recovery Codes
    backup_codes              TEXT[]            NOT NULL DEFAULT '{}',
    backup_codes_generated_at TIMESTAMPTZ,

    -- State Engine
    is_enabled                BOOLEAN           NOT NULL DEFAULT FALSE,
    enabled_at                TIMESTAMPTZ,
    last_verified_at          TIMESTAMPTZ,
    last_method_used          two_factor_method,

    -- Timestamps
    created_at                TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ       NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT two_factor_auth_pkey
        PRIMARY KEY (id),
    CONSTRAINT two_factor_auth_user_unique
        UNIQUE (user_id),
    CONSTRAINT fk_2fa_hookzen_user
        FOREIGN KEY (user_id)
        REFERENCES hookzen_users (id)
        ON DELETE CASCADE,
    CONSTRAINT chk_totp_digits
        CHECK (totp_digits IN (6, 8)),
    CONSTRAINT chk_totp_period
        CHECK (totp_period IN (30, 60)),
    CONSTRAINT chk_enabled_at
        CHECK (is_enabled = FALSE OR enabled_at IS NOT NULL)
);

CREATE INDEX idx_2fa_user_id
    ON two_factor_auth (user_id)
    WHERE is_enabled = TRUE;

-- ───────────────────────────────────────────────────────────────────
-- 5. TABLE: user_preferences
-- ───────────────────────────────────────────────────────────────────

CREATE TABLE user_preferences (
    user_id           UUID        PRIMARY KEY REFERENCES hookzen_users(id) ON DELETE CASCADE,
    
    -- Webhook Gateway Flattened Notifications (Direct clean mappings)
    notifications     JSONB       NOT NULL DEFAULT '{
        "delivery_failures": true,
        "endpoint_disabled": true,
        "rate_limit_warnings": true,
        "security_alerts": true
    }'::jsonb,

    -- System Preference Core
    language          VARCHAR(10) NOT NULL DEFAULT 'en',
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ───────────────────────────────────────────────────────────────────
-- 6. TRIGGER MAPPINGS (Using fn_set_updated_at from 000 migration)
-- ───────────────────────────────────────────────────────────────────

CREATE TRIGGER trg_hookzen_users_updated_at
    BEFORE UPDATE ON hookzen_users
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

CREATE TRIGGER trg_2fa_updated_at
    BEFORE UPDATE ON two_factor_auth
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

CREATE TRIGGER trg_user_preferences_updated_at
    BEFORE UPDATE ON user_preferences
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();