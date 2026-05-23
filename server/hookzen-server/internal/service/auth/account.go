package service

import (
	"context"

	"hookzen-server/internal/config"
	"hookzen-server/internal/dto"
	"hookzen-server/internal/email"
	"hookzen-server/internal/jwt"
	"hookzen-server/internal/model"
	"hookzen-server/internal/repository"

	sharedError "shared/errors"
	sharedLogging "shared/logging"
	sharedValidator "shared/validator"
	sharedSecurity "shared/security"
)

type AccountService struct {
	userRepo repository.UserRepository
	tokenRepo *repository.TokenRepository
	sessionRepo *repository.SessionRepository
	email *email.EmailService
	jwt *jwt.JWTManager
	cfg *config.Config
	logger *sharedLogging.Logger
	validator *sharedValidator.Validator
}

func NewAccountService(
	userRepo repository.UserRepository,
	tokenRepo *repository.TokenRepository,
	sessionRepo *repository.SessionRepository,
	email *email.EmailService,
	jwt *jwt.JWTManager,
	cfg *config.Config,
	logger *sharedLogging.Logger,
	validator *sharedValidator.Validator,
) *AccountService {
	return &AccountService {
		userRepo: userRepo,
		tokenRepo: tokenRepo,
		sessionRepo: sessionRepo,
		email: email,
		jwt: jwt,
		cfg: cfg,
		logger: logger,
		validator: validator,
	}
}

// =============================================================================
// ACCOUNT CREATION & VERIFICATION
// =============================================================================

func (a *AccountService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.MessageResponse, error) {

	if err := a.validator.ValidateStructToAppError(req); err != nil {
		return nil, err
	}

	exists, err := a.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, sharedError.BadRequest("User with this email already exists")
	}

	hash, err := sharedSecurity.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user, err := a.userRepo.Register(ctx, repository.RegisterParams{
		Email: req.Email,
		PasswordHash: hash,
		FullName: req.FullName,
	})
	if err != nil {
		return nil, err
	}

	if err := a.sendVerificationEmail(ctx, user); err != nil {
		a.logger.Error("Failed to send verification email", err , map[string]any{"user_id": user.ID})
	}

	return &dto.MessageResponse{
		Message: "Account created - please check your email to verify your account",
	}, nil
}

func (a *AccountService) ResendVerificationEmail(ctx context.Context, req *dto.ResendVerificationEmailRequest) (*dto.MessageResponse, error) {

    if err := a.validator.ValidateStructToAppError(req); err != nil {
        return nil, err
    }

    user, _, err := a.userRepo.GetByEmail(ctx, req.Email)
    if err != nil {
        if !sharedError.IsNotFound(err) {
            return nil, err
        }
    }
    
    if user == nil {
        return nil, sharedError.BadRequest("No account found with this email")
    }

    if err := a.sendVerificationEmail(ctx, user); err != nil {
        a.logger.Error("Failed to resend verification email", err, map[string]any{"user_id": user.ID})
        return nil, err 
    }

    return &dto.MessageResponse{
        Message: "Verification email has been resent successfully.",
    }, nil
}

func (a *AccountService) VerifyEmail(ctx context.Context, req *dto.VerifyEmailRequest) (*dto.AuthResponse, error) {
	userIDStr, err := a.tokenRepo.ConsumeEmailVerifyToken(ctx, req.Token)
	if err != nil {
		return nil, err
	}

	userID, err := sharedSecurity.ParseUUID(userIDStr)
	if err != nil {
		return nil, sharedError.BadRequest("Invalid token")
	}

	if err := a.userRepo.VerifyEmail(ctx, userID); err != nil {
		return nil, err
	}


}


// =============================================================================
// INTERNAL HELPERS
// =============================================================================

func (a *AccountService) sendVerificationEmail(ctx context.Context, user *model.HookzenUser) error {
    token, err := sharedSecurity.GenerateEmailVerifyToken()
    if err != nil {
        return sharedError.Wrap(err, 500, "Failed to generate email verification token")
    }

    ttl := a.cfg.Security.EmailVerificationTTL
    if err := a.tokenRepo.StoreEmailVerifyToken(ctx, user.ID.String(), token, ttl); err != nil {
        return sharedError.Wrap(err, 500, "Failed to store email verification token")
    }

    if err := a.email.SendVerificationEmail(ctx, user.Email, token); err != nil {
        return sharedError.Wrap(err, 500, "Failed to send verification email")
    }

    return nil
}