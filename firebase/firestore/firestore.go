// Package firestore implements helper functions and utilities to make working with package "cloud.google.com/go/firestore" easier.
package firestore

import (
	"context"
	"time"

	"github.com/d39b/kit/errors"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const firestoreErrOrigin = "firestore"

func NewFirestoreError(inner error, code errors.ErrorCode) errors.Error {
	return errors.New(inner, firestoreErrOrigin, code)
}

func NewFirestoreErrorInternal(inner error) errors.Error {
	return errors.New(inner, firestoreErrOrigin, errors.Internal)
}

// Parses the given firestore error and returns an instance of Error from package "github.com/d39b/kit/errors"
// with an appropriate error code set.
func ParseFirestoreError(err error) errors.Error {
	if status.Code(err) == codes.NotFound {
		return NewFirestoreError(err, errors.NotFound)
	} else {
		return NewFirestoreError(err, errors.Internal)
	}
}

// Unmarshal the given snapshot into result, which should usually be a pointer to a struct or map.
func UnmarshalDocSnapshot(snap *firestore.DocumentSnapshot, result interface{}) error {
	err := snap.DataTo(result)
	if err != nil {
		return NewFirestoreErrorInternal(err).WithInternalMessage("could not unmarshal document snapshot")
	}
	return nil
}

func GetDocumentSnapshotById(ctx context.Context, col *firestore.CollectionRef, id string) (*firestore.DocumentSnapshot, error) {
	snap, err := col.Doc(id).Get(ctx)
	if err != nil {
		return nil, ParseFirestoreError(err)
	}
	return snap, nil
}

// Gets and unmarshals the document with the given id into result.
func GetDocumentById(ctx context.Context, col *firestore.CollectionRef, id string, result interface{}) error {
	snap, err := GetDocumentSnapshotById(ctx, col, id)
	if err != nil {
		return err
	}
	return UnmarshalDocSnapshot(snap, result)
}

// Gets and unmarshals the document with the given id into result.
// Also returns a TransactionExpectations value that represents the last time the document
// was modified. Can be used to implement optimistic transactions.
func GetDocumentByIdWithTE(ctx context.Context, col *firestore.CollectionRef, id string, result interface{}) (TransactionExpectations, error) {
	snap, err := GetDocumentSnapshotById(ctx, col, id)
	if err != nil {
		return nil, err
	}
	err = UnmarshalDocSnapshot(snap, result)
	if err != nil {
		return nil, err
	}

	te := TransactionExpectations{}
	te.Add(TransactionExpectationFromSnapshot(snap))
	return te, nil
}

// Create a new document with the given id in the given collection.
func CreateDocument(ctx context.Context, col *firestore.CollectionRef, id string, doc interface{}) error {
	_, err := col.Doc(id).Create(ctx, doc)
	if err != nil {
		return ParseFirestoreError(err).WithInternalMessage("could not create document")
	}
	return nil
}

// Without merge options, can provide most data types (struct, map, slice, ...), document will be completely overridden.
// With "MergeAll" option, can only use map as data argument.
// With "Merge" option, only the provided fields will be overridden, can use structs as data argument.
func SetDocument(ctx context.Context, col *firestore.CollectionRef, id string, data interface{}, opts ...firestore.SetOption) error {
	_, err := col.Doc(id).Set(ctx, data, opts...)
	if err != nil {
		return ParseFirestoreError(err).WithInternalMessage("could not set document")
	}
	return nil
}

func UpdateDocument(ctx context.Context, col *firestore.CollectionRef, id string, updates []firestore.Update) error {
	_, err := col.Doc(id).Update(ctx, updates)
	if err != nil {
		return ParseFirestoreError(err).WithInternalMessage("could not update document")
	}
	return nil
}

func DeleteDocument(ctx context.Context, col *firestore.CollectionRef, id string) error {
	_, err := col.Doc(id).Delete(ctx)
	if err != nil {
		return ParseFirestoreError(err).WithInternalMessage("could not delete document")
	}
	return nil
}

func GetDocumentsForQuery(ctx context.Context, query firestore.Query) ([]*firestore.DocumentSnapshot, error) {
	docsnaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, ParseFirestoreError(err).WithInternalMessage("could not get documents for query")
	}
	return docsnaps, nil
}

// TransactionExpectation represents the state of a document, i.e. whether or not it exists and if it exists the last time it was updated/modified.
// Can be used to implement safe optimistic transactions.
type TransactionExpectation struct {
	DocRef     *firestore.DocumentRef
	Exists     bool
	UpdateTime time.Time
}

func TransactionExpectationFromSnapshot(snap *firestore.DocumentSnapshot) TransactionExpectation {
	return TransactionExpectation{
		DocRef:     snap.Ref,
		Exists:     snap.Exists(),
		UpdateTime: snap.UpdateTime,
	}
}

// Returns nil if the given document snapshot satisfies the transaction expectation, i.e.
// they both refer to the same document and existence as well as latest update times are equal.
func (te TransactionExpectation) IsSatisfied(snap *firestore.DocumentSnapshot) error {
	if te.DocRef.Path != snap.Ref.Path {
		return NewFirestoreError(nil, errors.InvalidArgument).
			WithInternalMessage("snapshot belongs to different document")
	}
	if te.Exists != snap.Exists() {
		return NewFirestoreError(nil, errors.FailedPrecondition).
			WithInternalMessage("transaction expectation failed: document existence changed")
	}
	if te.Exists {
		if !te.UpdateTime.Equal(snap.UpdateTime) {
			return NewFirestoreError(nil, errors.FailedPrecondition).
				WithInternalMessage("transaction expectation failed: document was updated")
		}
	}
	return nil
}

// A set of transaction expectations, that can be used to implement optimistic concurrency/transactions.
// Functions that read data from firestore can return TransactionExpectations values, multiple of them can be combined.
// If we then try to update the data and want to make sure that it hasn't changed since we last read it, we can use a transaction
// that first gets the documents and then compares them to the TransactionExpectations value using the VerifyTransactionExpectations() function.
//
// Optmistic concurrency works well if the probability of concurrent modifications is low.
// It has the advantage that it is easy to create code that guarantees consistency while not leaking any implementation details of the data store layer
// into business logic code.
type TransactionExpectations map[string]TransactionExpectation

func (tes TransactionExpectations) Add(te TransactionExpectation) {
	tes[te.DocRef.Path] = te
}

func (tes TransactionExpectations) Remove(docRef *firestore.DocumentRef) {
	delete(tes, docRef.Path)
}

// Combine two sets of transaction expectations.
// If both sets contain an expectation for the same document, the expectation with the more recent update time will be used.
func (tes TransactionExpectations) Combine(other TransactionExpectations) TransactionExpectations {
	combined := make(map[string]TransactionExpectation)
	for path, te := range tes {
		combined[path] = te
	}
	for path, te := range other {
		if e, ok := combined[path]; ok {
			if e.UpdateTime.Before(te.UpdateTime) {
				combined[path] = te
			}
		} else {
			combined[path] = te
		}
	}
	return combined
}

func (tes TransactionExpectations) Get(docRef *firestore.DocumentRef) (TransactionExpectation, bool) {
	te, ok := tes[docRef.Path]
	return te, ok
}

func (tes TransactionExpectations) DocRefs() []*firestore.DocumentRef {
	result := make([]*firestore.DocumentRef, len(tes))
	i := 0
	for _, te := range tes {
		result[i] = te.DocRef
		i++
	}
	return result
}

// Verifies that the given document snapshots are compatible/consistent with the set of transaction expectations, i.e.
// none of the documents were changed since the time the transaction expectations were created.
// Formally for each document snapshot it must be true that there is a transaction expectation for the same document
// and the document existence and latest update time of the snaphost and transaction expectation are equal.
//
// Note that there must be a transaction expectation for every snapshot, but not the other way around.
func (tes TransactionExpectations) Verify(snaps []*firestore.DocumentSnapshot) error {
	for _, snap := range snaps {
		te, ok := tes.Get(snap.Ref)
		if !ok {
			return NewFirestoreError(nil, errors.Internal).
				WithInternalMessage("missing transaction expectation, this might be a bug")
		}
		err := te.IsSatisfied(snap)
		if err != nil {
			return err
		}
	}
	return nil
}

// Verifies the transaction expectations in the given firestore transaction.
func VerifyTransactionExpectations(t *firestore.Transaction, te TransactionExpectations) error {
	docRefs := te.DocRefs()
	if len(docRefs) == 0 {
		return nil
	}

	snaps, err := t.GetAll(docRefs)
	if err != nil {
		return err
	}

	err = te.Verify(snaps)
	if err != nil {
		return err
	}
	return nil
}
