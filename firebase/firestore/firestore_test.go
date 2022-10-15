package firestore

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/d39b/kit/errors"
	"github.com/d39b/kit/firebase"
	"github.com/d39b/kit/firebase/emulator"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
)

// These tests require a running firestore emulator.
func TestMain(m *testing.M) {
	// Skip these tests if the environment variable is not set.
	if pid := os.Getenv("FIREBASE_PROJECT_ID"); pid == "" {
		log.Println("set FIREBASE_PROJECT_ID to run these tests")
		os.Exit(0)
	}

	exitCode := m.Run()
	os.Exit(exitCode)

}

// Creates a firestore emulator client that can be used to reset the emulator after each test and an instance of firestore.Client to implement the tests.
func initTest(t *testing.T) (*firestore.Client, error) {
	firestoreEmulatorClient, err := emulator.NewFirestoreEmulatorClient()
	if err != nil {
		t.Fatalf("could not create firestore emulator client: %v", err)
	}

	app, err := firebase.NewApp(firebase.Config{UseEmulators: true})
	if err != nil {
		t.Fatalf("could not create firebase app: %v", err)
	}
	ctx, cancel := getContext()
	defer cancel()
	firestore, err := app.Firestore(ctx)
	if err != nil {
		t.Fatalf("could not create firestore client: %v", err)
	}

	t.Cleanup(func() {
		err := firestoreEmulatorClient.ResetEmulator()
		if err != nil {
			t.Logf("could not reset firestore emulator: %v", err)
		}
	})

	return firestore, nil
}

func getContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Second)
}

type TestDoc struct {
	F1 string   `firestore:"f1"`
	F2 int      `firestore:"f2"`
	F3 []string `firestore:"f3"`
}

var testDoc TestDoc = TestDoc{
	F1: "just a string",
	F2: 42,
	F3: []string{"s1", "s3", "s5"},
}

func TestCreateAndGetDocumentWorks(t *testing.T) {
	a := assert.New(t)

	fs, err := initTest(t)
	a.Nil(err)

	col := fs.Collection("col1")
	id := "doc-1234-5678"

	ctx, cancel := getContext()
	defer cancel()

	err = CreateDocument(ctx, col, id, testDoc)
	a.Nil(err)

	var actualDoc TestDoc
	err = GetDocumentById(ctx, col, id, &actualDoc)
	a.Nil(err)
	a.Equal(testDoc, actualDoc)
}

func TestGetDocumentReturnsCorrectError(t *testing.T) {
	a := assert.New(t)

	fs, err := initTest(t)
	a.Nil(err)

	col := fs.Collection("col1")
	id := "doc-1234-5678"

	ctx, cancel := getContext()
	defer cancel()

	var actualDoc TestDoc
	err = GetDocumentById(ctx, col, id, &actualDoc)
	a.NotNil(err)
	a.True(errors.IsNotFoundError(err))
	a.Empty(actualDoc)
}

func TestDeleteDocumentWorks(t *testing.T) {
	a := assert.New(t)

	fs, err := initTest(t)
	a.Nil(err)

	col := fs.Collection("col1")
	id := "doc-1234-5678"

	ctx, cancel := getContext()
	defer cancel()

	err = CreateDocument(ctx, col, id, testDoc)
	a.Nil(err)

	err = DeleteDocument(ctx, col, id)
	a.Nil(err)

	var actualDoc TestDoc
	err = GetDocumentById(ctx, col, id, &actualDoc)
	a.NotNil(err)
	a.True(errors.IsNotFoundError(err))
	a.Empty(actualDoc)
}

func TestSetDocumentWorks(t *testing.T) {
	a := assert.New(t)

	fs, err := initTest(t)
	a.Nil(err)

	col := fs.Collection("col1")
	id := "doc-1234-5678"

	ctx, cancel := getContext()
	defer cancel()

	err = SetDocument(ctx, col, id, testDoc)
	a.Nil(err)

	err = SetDocument(ctx, col, id, map[string]interface{}{
		"f2": 1337,
	}, firestore.MergeAll)
	a.Nil(err)

	var actualDoc TestDoc
	err = GetDocumentById(ctx, col, id, &actualDoc)
	a.Nil(err)
	expectedDoc := testDoc
	expectedDoc.F2 = 1337
	a.Equal(expectedDoc, actualDoc)
}

func TestUpdateDocumentWorks(t *testing.T) {
	a := assert.New(t)

	fs, err := initTest(t)
	a.Nil(err)

	col := fs.Collection("col1")
	id := "doc-1234-5678"

	ctx, cancel := getContext()
	defer cancel()

	err = CreateDocument(ctx, col, id, testDoc)
	a.Nil(err)

	err = UpdateDocument(ctx, col, id, []firestore.Update{
		{Path: "f2", Value: 1337},
	})
	a.Nil(err)

	var actualDoc TestDoc
	err = GetDocumentById(ctx, col, id, &actualDoc)
	a.Nil(err)
	expectedDoc := testDoc
	expectedDoc.F2 = 1337
	a.Equal(expectedDoc, actualDoc)
}

func TestTransactionExpectations(t *testing.T) {
	a := assert.New(t)

	fs, err := initTest(t)
	a.Nil(err)

	col := fs.Collection("col1")
	id := "doc-1234-5678"

	ctx, cancel := getContext()
	defer cancel()

	err = CreateDocument(ctx, col, id, testDoc)
	a.Nil(err)

	snap, err := GetDocumentSnapshotById(ctx, col, id)
	a.Nil(err)
	te := TransactionExpectationFromSnapshot(snap)
	a.Equal(col.Doc(id), te.DocRef)
	a.True(te.Exists)
	a.Equal(snap.UpdateTime, te.UpdateTime)

	//get a new snapshot, transaction expectation should be satisfied
	snap, err = GetDocumentSnapshotById(ctx, col, id)
	a.Nil(err)
	a.Nil(te.IsSatisfied(snap))

	//Update the document and get a new snapshot
	err = UpdateDocument(ctx, col, id, []firestore.Update{
		{Path: "f2", Value: 1337},
	})
	a.Nil(err)
	snap, err = GetDocumentSnapshotById(ctx, col, id)
	a.Nil(err)
	a.NotNil(te.IsSatisfied(snap))

	//test a set of transaction expectations
	tes := TransactionExpectations{}

	id1 := "id1"
	id2 := "id2"

	err = CreateDocument(ctx, col, id1, testDoc)
	a.Nil(err)
	err = CreateDocument(ctx, col, id2, testDoc)
	a.Nil(err)

	snap, err = GetDocumentSnapshotById(ctx, col, id1)
	a.Nil(err)
	tes.Add(TransactionExpectationFromSnapshot(snap))
	snap, err = GetDocumentSnapshotById(ctx, col, id2)
	a.Nil(err)
	tes.Add(TransactionExpectationFromSnapshot(snap))

	//this should verify
	snaps, err := fs.GetAll(ctx, []*firestore.DocumentRef{col.Doc(id2), col.Doc(id1)})
	a.Nil(err)
	a.Nil(tes.VerifyTransactionExpectations(snaps))

	err = UpdateDocument(ctx, col, id1, []firestore.Update{
		{Path: "f2", Value: 1337},
	})
	a.Nil(err)

	//should not verify
	snaps, err = fs.GetAll(ctx, []*firestore.DocumentRef{col.Doc(id2), col.Doc(id1)})
	a.Nil(err)
	a.NotNil(tes.VerifyTransactionExpectations(snaps))
}
