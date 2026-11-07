package main

import (
	"context"
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"github.com/D4nRusu/band4band/backend/internal/database"
	"github.com/D4nRusu/band4band/backend/internal/httpapi"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment")
	}

	ctx := context.Background()

	app, db, err := database.InitFirebase(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	authClient, err := app.Auth(ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Listening on :8080")
	router := httpapi.NewRouter(
		authClient,
		database.NewUserStore(db),
		database.NewGroupStore(db),
		database.NewInvitationStore(db),
	)
	log.Fatal(http.ListenAndServe(":8080", router))
}
