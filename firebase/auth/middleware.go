package auth

import (
	"context"

	"github.com/d39b/kit/errors"

	kitjwt "github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/endpoint"
)

const authMwErrOrigin = "firebaseAuthMiddleware"

type ContextBuilderFunc func(context.Context, User) context.Context

// Endpoint middleware that uses an instance of AuthChecker to check if the request is authenticated, i.e.
// AuthChecker accepts the token obtained from the context using the key github.com/go-kit/kit/auth/jwt.JWTContextKey.
//
// If the token is valid the User value returned by the IsAuthenticated function of AuthChecker is passed into "ctxBuilder", which can return a new
// context that e.g. has the user stored as a value.
//
// If the request is not authenticated, the middleware will return an error immediately and not call the next endpoint handler.
func NewAuthEndpointMiddleware(ac AuthChecker, ctxBuilder ContextBuilderFunc) endpoint.Middleware {
	if ac == nil {
		panic("AuthChecker is nil")
	}
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			//if there is no value for the key, ctx.Value returns nil and the cast to string will fail
			token, ok := ctx.Value(kitjwt.JWTContextKey).(string)
			if !ok {
				return nil, errors.New(nil, authMwErrOrigin, errors.Unauthenticated).
					WithPublicMessage("no token provided")
			}
			user, err := ac.IsAuthenticated(ctx, token)
			if err != nil {
				return nil, err
			}
			var newCtx context.Context = ctx
			if ctxBuilder != nil {
				newCtx = ctxBuilder(ctx, user)
			}
			return next(newCtx, request)
		}
	}
}
