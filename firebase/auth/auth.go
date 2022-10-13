// Package auth provides utilities to integrate Firebase Authentication with the Go kit framework.
package auth

import (
	"context"

	"github.com/d39b/kit/errors"

	"firebase.google.com/go/v4/auth"
)

const ErrTokenExpired = 1
const ErrTokenInvalid = 2
const ErrEmailNotVerified = 3
const ErrInvalidCustomClaims = 4

const authCheckerErrOrigin = "authChecker"

type User struct {
	Uid          string
	CustomClaims interface{}
}

type ClaimsFunc func(map[string]interface{}) (interface{}, error)

// AuthChecker checks if a given JWT token is a valid Firebase Authentication token.
// If the token is valid, a User that contains the user id and custom claims is returned.
type AuthChecker interface {
	IsAuthenticated(ctx context.Context, token string) (User, error)
}

type authChecker struct {
	client               *auth.Client
	requireVerifiedEmail bool
	validateClaims       ClaimsFunc
}

// Returns a new instance of AuthChecker.
// If "requireVerifiedEmail" is true, the email of a user must be verified for a token to be considered valid.
// Furthermore a function to validate and extract custom claims of a user can be provided. A token will only be considered valid if this function returns a nil error.
// The first return value of the function is used to set the "CustomClaims" field of the User returned by the "IsAuthenticated" function.
func NewAuthChecker(authClient *auth.Client, requireVerifiedEmail bool, validateClaims ClaimsFunc) AuthChecker {
	return &authChecker{
		client:               authClient,
		requireVerifiedEmail: requireVerifiedEmail,
		validateClaims:       validateClaims,
	}
}

func (ac *authChecker) IsAuthenticated(ctx context.Context, token string) (User, error) {
	var user User
	verifiedToken, err := ac.client.VerifyIDToken(ctx, token)
	if err != nil {
		if auth.IsIDTokenExpired(err) {
			return user, errors.New(err, authCheckerErrOrigin, errors.Unauthenticated).
				WithPublicCode(ErrTokenExpired).
				WithPublicMessage("token expired")
		}
		if auth.IsIDTokenInvalid(err) {
			return user, errors.New(err, authCheckerErrOrigin, errors.Unauthenticated).
				WithPublicCode(ErrTokenInvalid).
				WithPublicMessage("token invalid")
		}
		return user, errors.New(err, authCheckerErrOrigin, errors.Internal)
	}
	uid := verifiedToken.UID
	if uid == "" {
		return user, errors.New(nil, authCheckerErrOrigin, errors.Internal).
			WithInternalMessage("uid empty, this shouldn't happen, probably a bug")
	}

	claims := verifiedToken.Claims

	if ac.requireVerifiedEmail {
		emailVerified, ok := claims["email_verified"].(bool)
		if !ok {
			return user, errors.New(nil, authCheckerErrOrigin, errors.Internal).
				WithInternalMessage("email_verified claim not found, this might be a bug")
		}
		if !emailVerified {
			return user, errors.New(nil, authCheckerErrOrigin, errors.Unauthenticated).
				WithPublicCode(ErrEmailNotVerified).
				WithPublicMessage("email not verified")
		}
	}

	if ac.validateClaims != nil {
		customClaims, err := ac.validateClaims(claims)
		if err != nil {
			return user, errors.New(nil, authCheckerErrOrigin, errors.Unauthenticated).
				WithPublicCode(ErrInvalidCustomClaims).
				WithPublicMessage("custom claims invalid")
		}
		user.CustomClaims = customClaims
	}

	user.Uid = uid
	return user, nil
}
