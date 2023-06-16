package main

import (
	"context"
	"log"

	"cloud.google.com/go/storage"
)

func main() {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	// check client access
	attrs, err := client.Bucket("peak-profits").Attrs(ctx)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	log.Println("success")
	log.Println(attrs)

}
