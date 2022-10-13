package auth

import (
	"context"
	stderrors "errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/d39b/kit/firebase"
	"github.com/d39b/kit/firebase/emulator"

	"github.com/d39b/kit/errors"

	"github.com/stretchr/testify/assert"
)

// These tests require a running firebase auth emulator.
func TestMain(m *testing.M) {
	// Skip these tests if the environment variable is not set.
	if pid := os.Getenv("FIREBASE_PROJECT_ID"); pid == "" {
		log.Println("set FIREBASE_PROJECT_ID to run these tests")
		os.Exit(0)
	}

	exitCode := m.Run()
	os.Exit(exitCode)

}

// Creates an auth emulator client that can be used to create test user accounts and an instance of AuthChecker that is tested.
func initTest(t *testing.T, requireVerifiedEmail bool, validateClaims ClaimsFunc) (*emulator.AuthEmulatorClient, AuthChecker, error) {
	authEmulatorClient, err := emulator.NewAuthEmulatorClient()
	if err != nil {
		t.Fatalf("could not create auth emulator client: %v", err)
	}

	app, err := firebase.NewApp(firebase.Config{UseEmulators: true})
	if err != nil {
		t.Fatalf("could not create firebase app: %v", err)
	}
	ctx, cancel := getContext()
	defer cancel()
	fbAuth, err := app.Auth(ctx)
	if err != nil {
		t.Fatalf("could not create firebase auth client: %v", err)
	}

	t.Cleanup(func() {
		err := authEmulatorClient.ResetEmulator()
		if err != nil {
			t.Logf("could not reset auth emulator: %v", err)
		}
	})

	return authEmulatorClient, NewAuthChecker(fbAuth, requireVerifiedEmail, validateClaims), nil
}

func getContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 1*time.Second)
}

func TestAuthCheckerWorksIfEmailVerificationNotRequired(t *testing.T) {
	a := assert.New(t)
	// AuthChecker will not require verified emails
	ec, ac, _ := initTest(t, false, nil)

	//user is not verified, should still be able to authenticate
	userId1, err := ec.CreateUser("test@test.de", "testpw1234", false)
	a.Nil(err)
	a.NotEmpty(userId1)
	token1, err := ec.SignInUser("test@test.de", "testpw1234")
	a.Nil(err)
	a.NotEmpty(token1)

	//user is verified, should also be authenticated
	userId2, err := ec.CreateUser("test1@test.de", "testpw1234", true)
	a.Nil(err)
	a.NotEmpty(userId2)
	token2, err := ec.SignInUser("test1@test.de", "testpw1234")
	a.Nil(err)
	a.NotEmpty(token2)

	ctx, cancel := getContext()
	defer cancel()
	user1, err := ac.IsAuthenticated(ctx, token1)
	a.Nil(err)
	a.Equal(userId1, user1.Uid)

	ctx1, cancel1 := getContext()
	defer cancel1()
	user2, err := ac.IsAuthenticated(ctx1, token2)
	a.Nil(err)
	a.Equal(userId2, user2.Uid)
}

func TestAuthCheckerWorksIfEmailVerificationRquired(t *testing.T) {
	a := assert.New(t)
	ec, ac, _ := initTest(t, true, nil)

	//user is not verified, should not be able to authenticate
	userId1, err := ec.CreateUser("test@test.de", "testpw1234", false)
	a.Nil(err)
	a.NotEmpty(userId1)
	token1, err := ec.SignInUser("test@test.de", "testpw1234")
	a.Nil(err)
	a.NotEmpty(token1)

	//user is verified, should be able to authenticate
	userId2, err := ec.CreateUser("test1@test.de", "testpw1234", true)
	a.Nil(err)
	a.NotEmpty(userId2)
	token2, err := ec.SignInUser("test1@test.de", "testpw1234")
	a.Nil(err)
	a.NotEmpty(token2)

	ctx, cancel := getContext()
	defer cancel()
	user1, err := ac.IsAuthenticated(ctx, token1)
	a.NotNil(err)
	a.True(errors.IsUnauthenticatedError(err))
	a.Empty(user1)

	ctx1, cancel1 := getContext()
	defer cancel1()
	user2, err := ac.IsAuthenticated(ctx1, token2)
	a.Nil(err)
	a.Equal(userId2, user2.Uid)
}

func TestAuthCheckerReturnsErrorOnInvalidToken(t *testing.T) {
	a := assert.New(t)
	_, ac, err := initTest(t, true, nil)

	//empty token should not be accepted
	ctx, cancel := getContext()
	defer cancel()
	user1, err := ac.IsAuthenticated(ctx, "")
	a.NotNil(err)
	a.True(errors.IsUnauthenticatedError(err))
	a.Equal(ErrTokenInvalid, err.(errors.Error).PublicCode)
	a.Empty(user1)

	ctx1, cancel1 := getContext()
	defer cancel1()
	user2, err := ac.IsAuthenticated(ctx1, "justsomerandomtoken")
	a.NotNil(err)
	a.True(errors.IsUnauthenticatedError(err))
	a.Equal(ErrTokenInvalid, err.(errors.Error).PublicCode)
	a.Empty(user2)
}

func TestAuthCheckerReturnsErrorIfCustomClaimsNotValid(t *testing.T) {
	a := assert.New(t)

	called := false
	validateClaimsFunc := func(m map[string]interface{}) (interface{}, error) {
		called = true
		return nil, stderrors.New("invalid claims")
	}

	ec, ac, err := initTest(t, true, validateClaimsFunc)

	userId1, err := ec.CreateUser("test@test.de", "testpw1234", true)
	a.Nil(err)
	a.NotEmpty(userId1)
	token1, err := ec.SignInUser("test@test.de", "testpw1234")
	a.Nil(err)
	a.NotEmpty(token1)

	ctx, cancel := getContext()
	defer cancel()
	user1, err := ac.IsAuthenticated(ctx, token1)
	a.NotNil(err)
	a.True(errors.IsUnauthenticatedError(err))
	a.Equal(ErrInvalidCustomClaims, err.(errors.Error).PublicCode)
	a.True(called)
	a.Empty(user1)
}

func TestAuthCheckerCustomClaimsAddedCorrectly(t *testing.T) {
	a := assert.New(t)

	called := false
	validateClaimsFunc := func(m map[string]interface{}) (interface{}, error) {
		called = true
		return "these_are_custom_claims", nil
	}

	ec, ac, err := initTest(t, true, validateClaimsFunc)

	userId1, err := ec.CreateUser("test@test.de", "testpw1234", true)
	a.Nil(err)
	a.NotEmpty(userId1)
	token1, err := ec.SignInUser("test@test.de", "testpw1234")
	a.Nil(err)
	a.NotEmpty(token1)

	ctx, cancel := getContext()
	defer cancel()
	user1, err := ac.IsAuthenticated(ctx, token1)
	a.Nil(err)
	a.True(called)
	a.Equal(User{Uid: userId1, CustomClaims: "these_are_custom_claims"}, user1)
}
