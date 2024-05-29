package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-http-utils/headers"
)

func main() {
	ackbarUrl := os.Getenv("ACKBAR_URL")
	contextId := os.Getenv("CONTEXT_ID")

	log.Printf("Starting worker in context %s", contextId)

	workersUrl := fmt.Sprintf("%s/contexts/%s/workers", ackbarUrl, contextId)

	log.Printf("Registering worker at %s", workersUrl)
	res, err := http.Post(workersUrl, "Content-Type:application/json", strings.NewReader(""))
	if err != nil {
		log.Fatalf("Failed registering worker: %e", err)
	}

	workerUrl := fmt.Sprintf("%s%s", ackbarUrl, res.Header.Get(headers.Location))
	log.Printf("Refresh URL is %s", workerUrl)

	log.Printf("Starting processing")
	defer res.Body.Close()

	for {
		time.Sleep(2 * time.Second)
		log.Printf("refreshing %s", workerUrl)
		req, err := http.NewRequest(http.MethodPut, workerUrl, nil)
		if err != nil {
			log.Fatalf("Failed creating refresh request: %e", err)
		}
		_, err = http.DefaultClient.Do(req)
		if err != nil {
			log.Fatalf("Failed to refresh worker: %e", err)
		}
	}
}
