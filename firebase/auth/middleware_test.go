package auth

import (
	"context"
	"testing"

	"github.com/d39b/kit/errors"

	kitjwt "github.com/go-kit/kit/auth/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAuthChecker struct {
	mock.Mock
}

func (m *MockAuthChecker) IsAuthenticated(ctx context.Context, token string) (User, error) {
	args := m.Called(token)
	return args.Get(0).(User), args.Error(1)
}

type contextKey string

const testContextKey contextKey = "testContextKey"

type contextValue struct {
	TestValue string
}

func TestAuthEndpointMiddlewareWorksIfUserAuthenticated(t *testing.T) {
	a := assert.New(t)

	ac := &MockAuthChecker{}
	user := User{Uid: "u-1234-5678", CustomClaims: nil}
	token := "justatoken"
	ac.On("IsAuthenticated", token).Return(user, nil)

	called := false
	var actualContextValue interface{}
	var actualContextValueOk bool
	ep := func(ctx context.Context, request interface{}) (interface{}, error) {
		//check that value was added to context correctly
		called = true
		actualContextValue, actualContextValueOk = ctx.Value(testContextKey).(contextValue)
		return nil, nil
	}
	ep = NewAuthEndpointMiddleware(ac, func(ctx context.Context, u User) context.Context {
		return context.WithValue(ctx, testContextKey, contextValue{TestValue: "just-a-test-value"})
	})(ep)

	ctx := context.WithValue(context.Background(), kitjwt.JWTContextKey, token)
	resp, err := ep(ctx, nil)
	a.Nil(resp)
	a.Nil(err)

	a.True(called)
	a.True(actualContextValueOk)
	a.Equal(contextValue{TestValue: "just-a-test-value"}, actualContextValue)
}

func TestAuthEndpointMiddlewareReturnsErrorIfNoTokenInContext(t *testing.T) {
	a := assert.New(t)
	ac := &MockAuthChecker{}

	called := false
	ep := func(ctx context.Context, request interface{}) (interface{}, error) {
		called = true
		return nil, nil
	}
	ep = NewAuthEndpointMiddleware(ac, nil)(ep)

	resp, err := ep(context.Background(), nil)
	a.Nil(resp)
	a.NotNil(err)
	a.True(errors.Is(err, errors.Unauthenticated))
	a.False(called)
}

func TestAuthEndpointMiddlewareReturnsErrorIfAuthCheckerReturnsError(t *testing.T) {
	a := assert.New(t)
	ac := &MockAuthChecker{}
	token := "justatoken"
	acErr := errors.New(nil, "mockAuthChecker", errors.Unauthenticated)
	ac.On("IsAuthenticated", token).Return(User{}, acErr)

	called := false
	ep := func(ctx context.Context, request interface{}) (interface{}, error) {
		called = true
		return nil, nil
	}
	ep = NewAuthEndpointMiddleware(ac, nil)(ep)

	ctx := context.WithValue(context.Background(), kitjwt.JWTContextKey, token)
	resp, err := ep(ctx, nil)
	a.Nil(resp)
	a.Equal(acErr, err)

	a.False(called)
}

func TestAuthEndpointMiddlewarePanicsIfAuthCheckerIsNil(t *testing.T) {
	a := assert.New(t)
	paniced := false
	func() {
		defer func() {
			if e := recover(); e != nil {
				paniced = true
			}
		}()
		NewAuthEndpointMiddleware(nil, nil)
	}()
	a.True(paniced)
}
