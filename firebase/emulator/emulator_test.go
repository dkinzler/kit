package emulator

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// These tests require running firebase emulators.
func TestMain(m *testing.M) {
	// Skip these tests if the environment variable is not set.
	// This is more convenient than using build tags, since having a build tag in the file
	// interferes with IDEs ability to compile the code and show errors/warnings/etc.
	// See also this discussion: https://peter.bourgon.org/blog/2021/04/02/dont-use-build-tags-for-integration-tests.html
	if pid := os.Getenv("FIREBASE_PROJECT_ID"); pid == "" {
		log.Println("set FIREBASE_PROJECT_ID to run these tests")
		os.Exit(0)
	}

	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestAuthEmulatorClient(t *testing.T) {
	a := assert.New(t)
	authEmulatorClient, err := NewAuthEmulatorClient()
	a.Nil(err)

	uid, err := authEmulatorClient.CreateUser("test@test.de", "testpw1234", true)
	a.Nil(err)
	t.Logf("Created user with id: %v", uid)

	token, err := authEmulatorClient.SignInUser("test@test.de", "testpw1234")
	a.Nil(err)
	t.Logf("Signed in user, token: %v", token)

	err = authEmulatorClient.ResetEmulator()
	a.Nil(err)
}

func TestFirestoreEmulatorClient(t *testing.T) {
	a := assert.New(t)

	firestoreEmulatorClient, err := NewFirestoreEmulatorClient()
	a.Nil(err)

	err = firestoreEmulatorClient.ResetEmulator()
	a.Nil(err)
}
