package main

import (
    "encoding/json"
    "log"
    "net/http"
)

type Response struct {
    Message string `json:"message"`
    Status  string `json:"status"`
}

func main() {
    mux := http.NewServeMux()

    mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")

        json.NewEncoder(w).Encode(Response{
            Message: "Backend is running",
            Status:  "ok",
        })
    })

    log.Println("Server listening on port 8080")

    log.Fatal(http.ListenAndServe(":8080", mux))
}