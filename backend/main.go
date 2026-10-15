package main

import (
    "context"
    "log"

    "github.com/D4nRusu/band4band/backend/internal/database"
)

func main() {
    ctx := context.Background()

    _, db, err := database.InitFirebase(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    _, err = db.Collection("test").Doc("connection").Set(ctx, map[string]interface{}{
        "message": "Hello from Go!",
        "working": true,
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Println("Firestore connection successful!")
}