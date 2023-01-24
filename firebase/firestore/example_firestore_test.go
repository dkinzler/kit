package firestore

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/dkinzler/kit/errors"
)

// An example of how to integrate optimistic concurrency/transactions with application services and data stores based on firestore.

type Folder struct {
	FolderId string
	Name     string
	// Ids of the notes contained in this folder
	Notes []string
}

func (f *Folder) ContainsNote(noteId string) bool {
	for _, n := range f.Notes {
		if n == noteId {
			return true
		}
	}
	return false
}

func (f *Folder) AddNote(noteId string) {
	f.Notes = append(f.Notes, noteId)
}

type Note struct {
	NoteId string
	Text   string
}

// Any type implementing this interface can be used as the data store for the note taking application.
// To implement optimistic concurrency/transactions, methods that read data return a transaction expectation (interface type TE).
// Methods that write/update data take a transaction expectation and must guarantee that the write is only performed
// if the transaction expectations are still satisfied (i.e. the data they represent was not modified since the time the transaction expectation was created).
type NoteDatastore interface {
	Folder(ctx context.Context, folderId string) (Folder, TE, error)
	Note(ctx context.Context, id string) (Note, TE, error)
	UpdateFolder(ctx context.Context, folderId string, f Folder, te TE) error
}

// Transaction expectations, usually a timestamp representing the last time some piece of data was modified.
type TE interface {
	Combine(other TE) TE
}

func ExampleTransactionExpectations() {
	// Imagine this is part of a method in a note taking application, where
	// we want to add a note to a folder. A folder can contain multiple notes.
	// To do this, we need to load the note and the folder from the data store.
	// Then we update the folder by adding the note to it and then persist
	// the changes by updating the folder in the data store.
	// We need to make sure that the data stays consistent, e.g. if a concurrent operation deleted the folder or the note,
	// the update operation on the data store should fail.
	// To this end we combine the transaction expectations for the folder and note
	// and pass them along to the UpdateFolder method of the data store.
	// The data store implementation can then make sure to only perform the update if the note and folder were not modified.
	//
	// firestoreNoteDatastore is an example implementation of NoteDatastore using firestore. It demonstrates how to use
	// transaction expectations to implement optimistic concurrency.

	ds := NewFirestoreNoteDatastore()

	folder, te1, _ := ds.Folder(context.Background(), "folder1234")
	note, te2, _ := ds.Note(context.Background(), "note1")

	if !folder.ContainsNote(note.NoteId) {
		folder.AddNote(note.NoteId)
	}

	te := te1.Combine(te2)

	ds.UpdateFolder(context.Background(), folder.FolderId, folder, te)
}

type firestoreNoteDatastore struct {
	client           *firestore.Client
	folderCollection *firestore.CollectionRef
	noteCollection   *firestore.CollectionRef
}

func NewFirestoreNoteDatastore() *firestoreNoteDatastore {
	// In an actual application we would have to provide a firestore client and initialize the collections.
	return &firestoreNoteDatastore{}
}

type firestoreTE TransactionExpectations

func (fte firestoreTE) Combine(other TE) TE {
	if ote, ok := other.(firestoreTE); ok {
		return fte.Combine(ote)
	}
	return fte
}

func (fds *firestoreNoteDatastore) Folder(ctx context.Context, folderId string) (Folder, TE, error) {
	var folder Folder
	te, err := GetDocumentByIdWithTE(ctx, fds.folderCollection, folderId, &folder)
	if err != nil {
		return Folder{}, nil, err
	}
	return folder, firestoreTE(te), nil
}

func (fds *firestoreNoteDatastore) Note(ctx context.Context, noteId string) (Note, TE, error) {
	var note Note
	te, err := GetDocumentByIdWithTE(ctx, fds.folderCollection, noteId, &note)
	if err != nil {
		return Note{}, nil, err
	}
	return note, firestoreTE(te), nil
}

func (fds *firestoreNoteDatastore) UpdateFolder(ctx context.Context, folderId string, f Folder, te TE) error {
	err := fds.client.RunTransaction(ctx, func(ctx context.Context, t *firestore.Transaction) error {
		if te != nil {
			// make sure the transaction expectation passed to the method has the correct type
			tmp, ok := te.(firestoreTE)
			if !ok {
				return NewFirestoreError(nil, errors.InvalidArgument)
			}
			fte := TransactionExpectations(tmp)

			// verify the transaction expectations, i.e. none of the documents have been modified
			if err := VerifyTransactionExpectations(t, fte); err != nil {
				return err
			}
		}

		err := t.Set(fds.folderCollection.Doc(folderId), f)
		if err != nil {
			return err
		}

		return nil
	}, firestore.MaxAttempts(1))
	return err
}
