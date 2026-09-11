package httpx

import (
	"errors"
	"reflect"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// ConfigureValidator makes the binding validator report JSON field names
// ("slug") instead of Go field names ("Slug"). Call once at startup.
func ConfigureValidator() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

// Pagination reads ?limit & ?offset with sane defaults and clamping.
func Pagination(c *gin.Context) (limit, offset int) {
	limit = defaultLimit
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
		limit = v
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if v, err := strconv.Atoi(c.Query("offset")); err == nil && v >= 0 {
		offset = v
	}
	return limit, offset
}

// UUIDParam parses a path parameter as a UUID, reporting a BAD_REQUEST error
// (already pushed onto the gin context) when it is malformed.
func UUIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		_ = c.Error(BadRequest("path parameter " + name + " must be a UUID"))
		return uuid.Nil, false
	}
	return id, true
}

// BindJSON binds and validates the request body, reporting a per-field
// VALIDATION_ERROR (or BAD_REQUEST for malformed JSON) on failure.
func BindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		_ = c.Error(bindingError(err))
		return false
	}
	return true
}

func bindingError(err error) *Error {
	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		fields := make([]FieldError, 0, len(validationErrs))
		for _, fe := range validationErrs {
			fields = append(fields, FieldError{Field: fe.Field(), Message: validationMessage(fe)})
		}
		return Validation("one or more fields are invalid").WithFields(fields)
	}
	return BadRequest("request body is not valid JSON").Wrap(err)
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "this field is required"
	case "email":
		return "must be a valid email address"
	case "uuid", "uuid4":
		return "must be a valid UUID"
	case "min":
		return "must be at least " + fe.Param() + " characters"
	case "max":
		return "must be at most " + fe.Param() + " characters"
	case "len":
		return "must be exactly " + fe.Param() + " characters"
	case "oneof":
		return "must be one of: " + strings.ReplaceAll(fe.Param(), " ", ", ")
	case "gte":
		return "must be " + fe.Param() + " or greater"
	case "lte":
		return "must be " + fe.Param() + " or less"
	case "url":
		return "must be a valid URL"
	default:
		return "failed the '" + fe.Tag() + "' rule"
	}
}
