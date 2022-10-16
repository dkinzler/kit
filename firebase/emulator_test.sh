export FIREBASE_PROJECT_ID="demo-project"
export FIREBASE_AUTH_EMULATOR_HOST="localhost:9099"
export FIRESTORE_EMULATOR_HOST="localhost:8080"

firebase -P "demo-project" -c ".firebase.json" emulators:exec "\
    go test -v -count=1 ./emulator && \
    go test -v -count=1 ./auth && \
    go test -v -count=1 ./firestore"