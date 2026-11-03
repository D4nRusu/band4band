package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"

	"github.com/D4nRusu/band4band/backend/internal/database"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment")
	}

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
