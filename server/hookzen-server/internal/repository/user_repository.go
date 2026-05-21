package repository

import (
	"context"
	"errors"

	"hookzen-server/internal/model"
	sharedErrs "shared/errors"
	"shared/logging"

	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─────────────────────────────────────────────────────────────────────────────
// REPOSITORY INTERFACE
// ─────────────────────────────────────────────────────────────────────────────

type UserRepository interface {
	Register(ctx context.Context, params RegisterParams) (*model.HookzenUser, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.HookzenUser, *model.UserPreferences, error)
	GetByEmail(ctx context.Context, email string) (*model.HookzenUser, *model.UserPreferences, error)
	VerifyEmail(ctx context.Context, userID uuid.UUID) error
	MarkPasswordResetRequested(ctx context.Context, userID uuid.UUID) error
	ResetPassword(ctx context.Context, userID uuid.UUID, newHash string) error
	Setup2FA(ctx context.Context, params Setup2FAParams) error
	Verify2FA(ctx context.Context, userID uuid.UUID, method model.TwoFactorMethod) error
	Disable2FA(ctx context.Context, userID uuid.UUID) error
	ConsumeBackupCode(ctx context.Context, userID uuid.UUID, remainingCodes []string) error
	GetTwoFactor(ctx context.Context, userID uuid.UUID) (*model.TwoFactorAuth, error)
	RecordLogin(ctx context.Context, userID uuid.UUID, ip string) error
	UpdateUser(ctx context.Context, id uuid.UUID, params UpdateUserParams) error
	UpdatePreferences(ctx context.Context, userID uuid.UUID, params UpdatePreferencesParams) error
	ListUsers(ctx context.Context, params ListUsersParams) ([]model.UserWithPrefs, int64, error)
}

// ─────────────────────────────────────────────────────────────────────────────
// PARAMETER STRUCTS
// ─────────────────────────────────────────────────────────────────────────────

type RegisterParams struct {
	Email        string
	PasswordHash string
	FullName     string
}

type Setup2FAParams struct {
	UserID            uuid.UUID
	EncryptedSecret   string
	HashedBackupCodes []string
}

type UpdateUserParams struct {
	FullName *string
	Timezone *string
	Locale   *string
}

type UpdatePreferencesParams struct {
	Notifications []byte // Raw JSONB parsing array
	Language      *string
}

type ListUsersParams struct {
	Search        string
	IsActive      *bool
	EmailVerified *bool
	Limit         int
	Offset        int
}

// ─────────────────────────────────────────────────────────────────────────────
// SQL COLUMN CONSTANTS (For Performance & Clean Queries)
// ─────────────────────────────────────────────────────────────────────────────

const userColumns = `
	id, email, email_verified, email_verified_at, password_hash, password_changed_at,
	password_reset_requested_at, full_name, timezone, locale, is_active,
	last_login_at, last_login_ip, two_factor_enabled, two_factor_enabled_at, created_at, updated_at, deleted_at`

const userWithPrefsColumns = `
	u.id, u.email, u.email_verified, u.email_verified_at, u.password_hash, u.password_changed_at,
	u.password_reset_requested_at, u.full_name, u.timezone, u.locale, u.is_active,
	u.last_login_at, u.last_login_ip, u.two_factor_enabled, u.two_factor_enabled_at, u.created_at, u.updated_at, u.deleted_at,
	p.user_id, p.notifications, p.language, p.updated_at`

const userJoinPrefs = `
	FROM hookzen_users u
	LEFT JOIN user_preferences p ON p.user_id = u.id`

// ─────────────────────────────────────────────────────────────────────────────
// REPOSITORY STRUCT & CONSTRUCTOR
// ─────────────────────────────────────────────────────────────────────────────

type userRepo struct {
	db *pgxpool.Pool // Dependency Injection: No global states!
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepo{db: db}
}

// ─────────────────────────────────────────────────────────────────────────────
// OPERATIONS IMPLEMENTATION
// ─────────────────────────────────────────────────────────────────────────────

func (r *userRepo) Register(ctx context.Context, params RegisterParams) (*model.HookzenUser, error) {
	var user model.HookzenUser

	// Managed ACID Transaction block
	err := r.withTransaction(ctx, func(tx pgx.Tx) error {
		const sql = `
			INSERT INTO hookzen_users (email, password_hash, full_name)
			VALUES (@email, @password_hash, @full_name)
			RETURNING ` + userColumns

		args := pgx.NamedArgs{
			"email":         params.Email,
			"password_hash": params.PasswordHash,
			"full_name":     params.FullName,
		}

		rows, err := tx.Query(ctx, sql, args)
		if err != nil {
			return err
		}
		
		user, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[model.HookzenUser])
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `INSERT INTO user_preferences (user_id) VALUES ($1)`, user.ID)
		return err
	})

	if err != nil {
		if isUniqueViolation(err) {
			return nil, sharedErrs.Conflict("email is already registered")
		}
		logging.Error("repo: register transaction failed", err, nil)
		return nil, sharedErrs.Internal(err)
	}

	return &user, nil
}

func (r *userRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.HookzenUser, *model.UserPreferences, error) {
	sql := `SELECT ` + userWithPrefsColumns + userJoinPrefs + ` WHERE u.id = @id AND u.deleted_at IS NULL`

	rows, err := r.db.Query(ctx, sql, pgx.NamedArgs{"id": id})
	if err != nil {
		logging.Error("repo: get user by id failed", err, map[string]any{"id": id})
		return nil, nil, sharedErrs.Internal(err)
	}

	res, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.UserWithPrefs])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, sharedErrs.NotFound("user entity not found")
		}
		return nil, nil, sharedErrs.Internal(err)
	}

	return &res.User, &res.Preferences, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*model.HookzenUser, *model.UserPreferences, error) {
	sql := `SELECT ` + userWithPrefsColumns + userJoinPrefs + ` WHERE u.email = @email AND u.deleted_at IS NULL`

	rows, err := r.db.Query(ctx, sql, pgx.NamedArgs{"email": email})
	if err != nil {
		logging.Error("repo: get user by email query failed", err, nil)
		return nil, nil, sharedErrs.Internal(err)
	}

	res, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.UserWithPrefs])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, sharedErrs.NotFound("invalid email identity or password request bounds")
		}
		return nil, nil, sharedErrs.Internal(err)
	}

	return &res.User, &res.Preferences, nil
}

func (r *userRepo) VerifyEmail(ctx context.Context, userID uuid.UUID) error {
	sql := `
		UPDATE hookzen_users
		SET    email_verified    = TRUE,
		       email_verified_at = NOW(),
		       updated_at        = NOW()
		WHERE  id = @id AND deleted_at IS NULL AND email_verified = FALSE`

	tag, err := r.db.Exec(ctx, sql, pgx.NamedArgs{"id": userID})
	if err != nil {
		logging.Error("repo: verify email execution crashed", err, map[string]any{"id": userID})
		return sharedErrs.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return sharedErrs.NotFound("user not found or already verified")
	}
	return nil
}

func (r *userRepo) MarkPasswordResetRequested(ctx context.Context, userID uuid.UUID) error {
	sql := `
		UPDATE hookzen_users
		SET password_reset_requested_at = NOW(), updated_at = NOW()
		WHERE id = @id AND deleted_at IS NULL`

	tag, err := r.db.Exec(ctx, sql, pgx.NamedArgs{"id": userID})
	if err != nil {
		return sharedErrs.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return sharedErrs.NotFound("user not found")
	}
	return nil
}

func (r *userRepo) ResetPassword(ctx context.Context, userID uuid.UUID, newHash string) error {
	sql := `
		UPDATE hookzen_users
		SET password_hash                = @hash,
		    password_changed_at          = NOW(),
		    password_reset_requested_at  = NULL,
		    updated_at                   = NOW()
		WHERE id = @id AND deleted_at IS NULL`

	tag, err := r.db.Exec(ctx, sql, pgx.NamedArgs{"id": userID, "hash": newHash})
	if err != nil {
		return sharedErrs.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return sharedErrs.NotFound("user modification execution target missing")
	}
	return nil
}

func (r *userRepo) Setup2FA(ctx context.Context, params Setup2FAParams) error {
	sql := `
		INSERT INTO two_factor_auth (user_id, totp_secret, backup_codes, backup_codes_generated_at)
		VALUES (@user_id, @totp_secret, @backup_codes, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			totp_secret               = EXCLUDED.totp_secret,
			backup_codes              = EXCLUDED.backup_codes,
			backup_codes_generated_at = NOW(),
			is_enabled                = FALSE,
			enabled_at                = NULL`

	_, err := r.db.Exec(ctx, sql, pgx.NamedArgs{
		"user_id":      params.UserID,
		"totp_secret":  params.EncryptedSecret,
		"backup_codes": params.HashedBackupCodes,
	})
	if err != nil {
		logging.Error("repo: multi-factor structural setup aborted", err, map[string]any{"id": params.UserID})
		return sharedErrs.Internal(err)
	}
	return nil
}

func (r *userRepo) Verify2FA(ctx context.Context, userID uuid.UUID, method model.TwoFactorMethod) error {
	return r.withTransaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE two_factor_auth
			SET is_enabled = TRUE, enabled_at = COALESCE(enabled_at, NOW()), last_verified_at = NOW(), last_method_used = @method
			WHERE user_id = @user_id`,
			pgx.NamedArgs{"user_id": userID, "method": method},
		)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, `
			UPDATE hookzen_users
			SET two_factor_enabled = TRUE, two_factor_enabled_at = COALESCE(two_factor_enabled_at, NOW()), updated_at = NOW()
			WHERE id = @id`,
			pgx.NamedArgs{"id": userID},
		)
		return err
	})
}

func (r *userRepo) Disable2FA(ctx context.Context, userID uuid.UUID) error {
	return r.withTransaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE two_factor_auth SET is_enabled = FALSE, enabled_at = NULL WHERE user_id = $1`, userID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE hookzen_users SET two_factor_enabled = FALSE, two_factor_enabled_at = NULL, updated_at = NOW() WHERE id = $1`, userID)
		return err
	})
}

func (r *userRepo) ConsumeBackupCode(ctx context.Context, userID uuid.UUID, remainingCodes []string) error {
	_, err := r.db.Exec(ctx, `UPDATE two_factor_auth SET backup_codes = $1, updated_at = NOW() WHERE user_id = $2`, remainingCodes, userID)
	if err != nil {
		return sharedErrs.Internal(err)
	}
	return nil
}

func (r *userRepo) GetTwoFactor(ctx context.Context, userID uuid.UUID) (*model.TwoFactorAuth, error) {
	sql := `
		SELECT id, user_id, totp_secret, totp_algorithm, totp_digits, totp_period, backup_codes, backup_codes_generated_at, is_enabled, enabled_at, last_verified_at, last_method_used, created_at, updated_at
		FROM two_factor_auth WHERE user_id = $1`

	rows, err := r.db.Query(ctx, sql, userID)
	if err != nil {
		return nil, sharedErrs.Internal(err)
	}
	tf, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.TwoFactorAuth])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, sharedErrs.NotFound("2FA configuration profile empty")
		}
		return nil, sharedErrs.Internal(err)
	}
	return &tf, nil
}

func (r *userRepo) RecordLogin(ctx context.Context, userID uuid.UUID, ip string) error {
	sql := `UPDATE hookzen_users SET last_login_at = NOW(), last_login_ip = @ip WHERE id = @id AND deleted_at IS NULL`
	_, err := r.db.Exec(ctx, sql, pgx.NamedArgs{"id": userID, "ip": ip})
	if err != nil {
		logging.Error("repo: login state update failed", err, map[string]any{"id": userID})
		return sharedErrs.Internal(err)
	}
	return nil
}

func (r *userRepo) UpdateUser(ctx context.Context, id uuid.UUID, params UpdateUserParams) error {
	sql := `
		UPDATE hookzen_users
		SET full_name = COALESCE(@full_name, full_name),
		    timezone  = COALESCE(@timezone, timezone),
		    locale    = COALESCE(@locale, locale),
		    updated_at = NOW()
		WHERE id = @id AND deleted_at IS NULL`

	tag, err := r.db.Exec(ctx, sql, pgx.NamedArgs{
		"id":        id,
		"full_name": params.FullName,
		"timezone":  params.Timezone,
		"locale":    params.Locale,
	})
	if err != nil {
		return sharedErrs.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return sharedErrs.NotFound("user identity targets not modified")
	}
	return nil
}

func (r *userRepo) UpdatePreferences(ctx context.Context, userID uuid.UUID, params UpdatePreferencesParams) error {
	sql := `
		UPDATE user_preferences
		SET notifications = COALESCE(@notifications, notifications),
		    language      = COALESCE(@language, language),
		    updated_at    = NOW()
		WHERE user_id = @user_id`

	var notifArg any = nil
	if len(params.Notifications) > 0 {
		notifArg = params.Notifications
	}

	tag, err := r.db.Exec(ctx, sql, pgx.NamedArgs{
		"user_id":       userID,
		"notifications": notifArg,
		"language":      params.Language,
	})
	if err != nil {
		return sharedErrs.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return sharedErrs.NotFound("preferences structure target missed")
	}
	return nil
}

func (r *userRepo) ListUsers(ctx context.Context, params ListUsersParams) ([]model.UserWithPrefs, int64, error) {
	selectSQL := `SELECT ` + userWithPrefsColumns + userJoinPrefs + ` WHERE u.deleted_at IS NULL`
	countSQL := `SELECT COUNT(*) FROM hookzen_users u WHERE u.deleted_at IS NULL`

	whereClause := ""
	args := pgx.NamedArgs{}

	if params.Search != "" {
		whereClause += ` AND (u.email ILIKE @search OR u.full_name ILIKE @search)`
		args["search"] = "%" + params.Search + "%"
	}
	if params.IsActive != nil {
		whereClause += ` AND u.is_active = @is_active`
		args["is_active"] = *params.IsActive
	}
	if params.EmailVerified != nil {
		whereClause += ` AND u.email_verified = @email_verified`
		args["email_verified"] = *params.EmailVerified
	}

	var total int64
	err := r.db.QueryRow(ctx, countSQL+whereClause, args).Scan(&total)
	if err != nil {
		return nil, 0, sharedErrs.Internal(err)
	}

	finalSQL := selectSQL + whereClause + ` ORDER BY u.created_at DESC`
	if params.Limit > 0 {
		finalSQL += ` LIMIT @limit`
		args["limit"] = params.Limit
	}
	if params.Offset > 0 {
		finalSQL += ` OFFSET @offset`
		args["offset"] = params.Offset
	}

	rows, err := r.db.Query(ctx, finalSQL, args)
	if err != nil {
		return nil, 0, sharedErrs.Internal(err)
	}
	defer rows.Close()

	res, err := pgx.CollectRows(rows, pgx.RowToStructByName[model.UserWithPrefs])
	if err != nil {
		return nil, 0, sharedErrs.Internal(err)
	}

	return res, total, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// COMPONENT TRANSACTION/CONSTRAINT PRIVATE HANDLERS
// ─────────────────────────────────────────────────────────────────────────────

func (r *userRepo) withTransaction(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}