// Package emulator provides helpers for working with firebase emulators, e.g. to populate them with data or reset them.
// This package is intended to be mostly used in tests, since the emulators are for testing as well.
package emulator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	p "path"
	"time"

	"github.com/d39b/kit/firebase"

	"firebase.google.com/go/v4/auth"
)

// AuthEmulatorClient can be used to initialize and reset the firebase auth emulator for tests.
// It provides functions to create uesrs, sign a user in and obtain a JWT token, and reset the emulator by deleting all accounts.
type AuthEmulatorClient struct {
	projectId string
	//these are used to directly interact with the emulator over http
	emulatorAddress  string
	emulatorBasePath string
	//used to create users
	fbAuthClient *auth.Client
}

func NewAuthEmulatorClient() (*AuthEmulatorClient, error) {
	projectId := os.Getenv("FIREBASE_PROJECT_ID")
	if projectId == "" {
		return nil, errors.New("error: set FIREBASE_PROJECT_ID environment variable")
	}
	authEmulatorAddress := os.Getenv("FIREBASE_AUTH_EMULATOR_HOST")
	if authEmulatorAddress == "" {
		return nil, errors.New("error: set FIREBASE_AUTH_EMULATOR_HOST environment variable")
	}

	app, err := firebase.NewApp(firebase.Config{
		UseEmulators: true,
		ProjectId:    projectId,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	fbAuthClient, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not create firebase auth client: %w", err)
	}

	return &AuthEmulatorClient{
		projectId:        projectId,
		emulatorAddress:  authEmulatorAddress,
		emulatorBasePath: "/identitytoolkit.googleapis.com/v1",
		fbAuthClient:     fbAuthClient,
	}, nil
}

// Create a new user in the emulator.
// Returns the user id of the new user.
func (a *AuthEmulatorClient) CreateUser(email, password string, emailVerified bool) (string, error) {
	user := (&auth.UserToCreate{}).
		Email(email).
		Password(password).
		EmailVerified(emailVerified)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ur, err := a.fbAuthClient.CreateUser(ctx, user)
	if err != nil {
		return "", fmt.Errorf("could not create auth user: %w", err)
	}
	return ur.UID, nil
}

func IsEmailAlreadyExistsError(err error) bool {
	if inner := errors.Unwrap(err); inner != nil {
		return auth.IsEmailAlreadyExists(inner)
	}
	return auth.IsEmailAlreadyExists(err)
}

// Tries to obtain a JWT token from the emulator using the provided email and password.
func (a *AuthEmulatorClient) SignInUser(email, password string) (string, error) {
	u := buildUrl(a.emulatorAddress, a.emulatorBasePath, "accounts:signInWithPassword", map[string]string{
		"key": "fake-api-key",
	})
	body, err := encodeJSON(map[string]interface{}{
		"email":             email,
		"password":          password,
		"returnSecureToken": true,
	})
	if err != nil {
		return "", fmt.Errorf("could not encode json: %w", err)
	}

	resp, err := http.Post(u, contentTypeJSON, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("http request error: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("response has wrong status code: %v", resp.StatusCode)
	}

	//read JWT token from response
	var rb map[string]interface{}
	err = decodeJSON(resp.Body, &rb)
	if err != nil {
		return "", fmt.Errorf("could not decode json response body: %w", err)
	}
	token, ok := rb["idToken"].(string)
	if !ok {
		return "", fmt.Errorf("token not found in response: %w", err)
	}
	return token, nil
}

// Reset firebase auth emulator by deleting all user accounts.
func (a *AuthEmulatorClient) ResetEmulator() error {
	client := http.Client{}
	req, err := http.NewRequest(
		http.MethodDelete,
		buildUrl(a.emulatorAddress, fmt.Sprintf("emulator/v1/projects/%v/accounts", a.projectId), "", nil),
		nil,
	)
	if err != nil {
		return fmt.Errorf("could not create http request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("response has wrong status code: %v", resp.StatusCode)
	}
	return nil
}

// FirestoreEmulatorClient can be used to reset the firestore emulator by deleting all data.
// When testing, we usually want to reset the emulator after every test, so that data created by one cannot influence another.
type FirestoreEmulatorClient struct {
	projectId string
	//these are used to directly interact with the emulator over http
	emulatorAddress  string
	emulatorBasePath string
}

func NewFirestoreEmulatorClient() (*FirestoreEmulatorClient, error) {
	projectId := os.Getenv("FIREBASE_PROJECT_ID")
	if projectId == "" {
		return nil, errors.New("error: set FIREBASE_PROJECT_ID environment variable")
	}
	emulatorAddress := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorAddress == "" {
		return nil, errors.New("error: set FIRESTORE_EMULATOR_HOST environment variable")
	}
	emulatorBasePath := "/emulator/v1/projects/" + projectId

	return &FirestoreEmulatorClient{
		projectId:        projectId,
		emulatorAddress:  emulatorAddress,
		emulatorBasePath: emulatorBasePath,
	}, nil
}

func (e *FirestoreEmulatorClient) ResetEmulator() error {
	client := http.Client{}
	req, err := http.NewRequest(
		http.MethodDelete,
		buildUrl(e.emulatorAddress, e.emulatorBasePath, "databases/(default)/documents", nil),
		nil,
	)
	if err != nil {
		return fmt.Errorf("could not create http request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request error: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("response has wrong status code: %v", resp.StatusCode)
	}
	return nil
}

func buildUrl(host, base, path string, queryParameters map[string]string) string {
	u := url.URL{}
	u.Scheme = "http"
	u.Host = host
	u.Path = p.Join(base, path)
	if queryParameters != nil {
		q := u.Query()
		for k, v := range queryParameters {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}
	return u.String()
}

const contentTypeJSON = "application/json"

func encodeJSON(b interface{}) ([]byte, error) {
	return json.Marshal(b)
}

func decodeJSON(body io.Reader, r interface{}) error {
	return json.NewDecoder(body).Decode(r)
}
