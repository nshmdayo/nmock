package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	fmt.Println("Starting server on port :9000")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome to my website!")
		fmt.Println("Successfully served request " + time.Now().Format(time.RFC3339))
	})

	http.ListenAndServe(":9000", nil)
}
