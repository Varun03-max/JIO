package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "5001"
    }
    addr := fmt.Sprintf("0.0.0.0:%s", port)

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "✅ JioTV Go server is running!")
    })

    log.Printf("Listening on %s", addr)
    log.Fatal(http.ListenAndServe(addr, nil))
}
