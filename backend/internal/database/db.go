package database

import (
    "context"

    firebase "firebase.google.com/go/v4"
    "cloud.google.com/go/firestore"
)

func InitFirebase(ctx context.Context) (*firebase.App, *firestore.Client, error) {
    app, err := firebase.NewApp(ctx, nil)
    if err != nil {
        return nil, nil, err
    }

    db, err := app.Firestore(ctx)
    if err != nil {
        return nil, nil, err
    }

    return app, db, nil
}