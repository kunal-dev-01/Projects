package validator

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	sharedErrs "shared/errors" 

	"github.com/go-playground/validator/v10"
	"github.com/shopspring/decimal"
)

var (
	validate *validator.Validate
	once     sync.Once
)

func GetValidator() *validator.Validate {
	once.Do(func() {
		validate = validator.New()
		registerCustomValidations(validate)
	})
	return validate
}

/* ───────────────── Pre-Compiled Regular Expressions ───────────────── */

var (
	nameRegex     = regexp.MustCompile(`^[a-zA-Z\s]{2,50}$`)
	phoneRegex    = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	slugRegex     = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	hasLower      = regexp.MustCompile(`[a-z]`)
	hasUpper      = regexp.MustCompile(`[A-Z]`)
	hasNumber     = regexp.MustCompile(`[0-9]`)
	hasSpecial    = regexp.MustCompile(`[!@#$%^&*]`)
	hasWhitespace = regexp.MustCompile(`\s`)
)

/* ───────────────── Custom Core Field Validations ───────────────── */

func registerCustomValidations(v *validator.Validate) {
	v.RegisterValidation("secure_password", func(fl validator.FieldLevel) bool {
		return IsStrongPassword(fl.Field().String())
	})

	v.RegisterValidation("valid_name", func(fl validator.FieldLevel) bool {
		return IsValidName(fl.Field().String())
	})

	v.RegisterValidation("phone", func(fl validator.FieldLevel) bool {
		return IsValidPhone(fl.Field().String())
	})

	v.RegisterValidation("slug", func(fl validator.FieldLevel) bool {
		return IsValidSlug(fl.Field().String())
	})

	v.RegisterValidation("no_space", func(fl validator.FieldLevel) bool {
		return !hasWhitespace.MatchString(fl.Field().String())
	})

	v.RegisterValidation("d_gte_zero", func(fl validator.FieldLevel) bool {
		d, ok := toDecimal(fl.Field())
		return ok && d.GreaterThanOrEqual(decimal.Zero)
	})

	v.RegisterValidation("d_gt_zero", func(fl validator.FieldLevel) bool {
		d, ok := toDecimal(fl.Field())
		return ok && d.GreaterThan(decimal.Zero)
	})

	v.RegisterValidation("d_gte", func(fl validator.FieldLevel) bool {
		d, ok := toDecimal(fl.Field())
		if !ok {
			return false
		}
		param := fl.Param()
		if param == "" {
			return d.GreaterThanOrEqual(decimal.Zero)
		}
		p, err := decimal.NewFromString(param)
		if err != nil {
			return false
		}
		return d.GreaterThanOrEqual(p)
	})

	v.RegisterValidation("d_gt_field", func(fl validator.FieldLevel) bool {
		d, ok := toDecimal(fl.Field())
		if !ok {
			return false
		}
		otherD, ok := getDecimalField(fl, fl.Param())
		if !ok {
			return false
		}
		return d.GreaterThan(otherD)
	})

	v.RegisterValidation("i_gte", func(fl validator.FieldLevel) bool {
		val, err := strconv.ParseInt(fmt.Sprintf("%v", fl.Field().Interface()), 10, 64)
		if err != nil {
			return false
		}
		param, err := strconv.ParseInt(fl.Param(), 10, 64)
		if err != nil {
			return false
		}
		return val >= param
	})

	v.RegisterValidation("i_lte", func(fl validator.FieldLevel) bool {
		val, err := strconv.ParseInt(fmt.Sprintf("%v", fl.Field().Interface()), 10, 64)
		if err != nil {
			return false
		}
		param, err := strconv.ParseInt(fl.Param(), 10, 64)
		if err != nil {
			return false
		}
		return val <= param
	})
}

/* ───────────────── Struct Evaluators ───────────────── */

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
	Fields map[string]string `json:"fields,omitempty"`
}

func ValidateStruct(s interface{}) *ValidationResult {
	err := GetValidator().Struct(s)
	if err == nil {
		return &ValidationResult{Valid: true}
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return &ValidationResult{
			Valid:  false,
			Errors: []ValidationError{{Field: "general", Message: "Invalid validation error format"}},
		}
	}

	result := &ValidationResult{
		Valid:  false,
		Errors: make([]ValidationError, 0, len(validationErrors)),
		Fields: make(map[string]string, len(validationErrors)),
	}

	for _, err := range validationErrors {
		field := strings.ToLower(err.Field())
		msg := formatErrorMessage(err)

		result.Errors = append(result.Errors, ValidationError{
			Field:   field,
			Message: msg,
		})
		result.Fields[field] = msg
	}

	return result
}

// ValidateStructToAppError processes mapping cleanly into shared domain error payloads
func ValidateStructToAppError(s interface{}) *sharedErrs.AppError {
	result := ValidateStruct(s)
	if result.Valid {
		return nil
	}
	return sharedErrs.Validation("Input validation failed", result.Fields)
}

func ValidateVar(field interface{}, tag string) error {
	return GetValidator().Var(field, tag)
}

/* ───────────────── Primitives Pure Checkers ───────────────── */

func IsStrongPassword(password string) bool {
	if len(password) < 8 || len(password) > 72 {
		return false
	}
	return hasLower.MatchString(password) &&
		hasUpper.MatchString(password) &&
		hasNumber.MatchString(password) &&
		hasSpecial.MatchString(password)
}

func IsValidEmail(email string) bool {
	return ValidateVar(email, "required,email") == nil
}

func IsValidName(name string) bool { return nameRegex.MatchString(name) }
func IsValidPhone(phone string) bool { return phoneRegex.MatchString(phone) }

func IsValidURL(input string) bool {
	if input == "" {
		return false
	}
	u, err := url.ParseRequestURI(input)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func IsValidAvatar(avatarURL string) bool {
	if !IsValidURL(avatarURL) {
		return false
	}
	lower := strings.ToLower(avatarURL)
	return strings.HasSuffix(lower, ".jpg") ||
		strings.HasSuffix(lower, ".jpeg") ||
		strings.HasSuffix(lower, ".png") ||
		strings.HasSuffix(lower, ".webp")
}

func IsValidSlug(slug string) bool { return slugRegex.MatchString(slug) && len(slug) <= 100 }
func IsPositiveInt(n int64) bool  { return n > 0 }
func IsInRange(n, min, max int64) bool { return n >= min && n <= max }
func IsValidPort(port int) bool    { return port > 0 && port <= 65535 }

/* ───────────────── Internal Parsing Engines ───────────────── */

func toDecimal(field reflect.Value) (decimal.Decimal, bool) {
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			return decimal.Zero, false
		}
		field = field.Elem()
	}
	d, ok := field.Interface().(decimal.Decimal)
	return d, ok
}

func getDecimalField(fl validator.FieldLevel, fieldName string) (decimal.Decimal, bool) {
	parent := fl.Parent()
	if parent.Kind() == reflect.Ptr {
		parent = parent.Elem()
	}
	otherField := parent.FieldByName(fieldName)
	if !otherField.IsValid() {
		return decimal.Zero, false
	}
	return toDecimal(otherField)
}

func formatErrorMessage(err validator.FieldError) string {
	tag := err.Tag()
	param := err.Param()

	switch tag {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email address format"
	case "secure_password":
		return "Password must include uppercase, lowercase, number, and a special character"
	case "valid_name":
		return "Name must only contain letters and spaces (2-50 characters)"
	case "phone":
		return "Invalid phone number format"
	case "slug":
		return "Invalid slug format (use lowercase letters, numbers, and hyphens only)"
	case "no_space":
		return "This field must not contain spaces"
	case "min":
		return fmt.Sprintf("Must be at least %s characters", param)
	case "max":
		return fmt.Sprintf("Cannot exceed %s characters", param)
	case "gte":
		return fmt.Sprintf("Must be at least %s", param)
	case "lte":
		return fmt.Sprintf("Must be at most %s", param)
	case "gt":
		return fmt.Sprintf("Must be greater than %s", param)
	case "lt":
		return fmt.Sprintf("Must be less than %s", param)
	case "url":
		return "Invalid URL format"
	case "oneof":
		return fmt.Sprintf("Must be one of: %s", strings.ReplaceAll(param, " ", ", "))
	case "d_gte_zero":
		return "Must be zero or greater"
	case "d_gt_zero":
		return "Must be greater than zero"
	case "d_gte":
		return fmt.Sprintf("Must be at least %s", param)
	case "d_gt_field":
		return fmt.Sprintf("Must be greater than %s", strings.ToLower(param))
	case "i_gte":
		return fmt.Sprintf("Must be at least %s", param)
	case "i_lte":
		return fmt.Sprintf("Must be at most %s", param)
	case "numeric", "number":
		return "Must be a valid number"
	case "len":
		return fmt.Sprintf("Must be exactly %s characters", param)
	default:
		return fmt.Sprintf("Validation failed on rule: %s", tag)
	}
}