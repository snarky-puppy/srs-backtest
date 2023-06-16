package pp

import (
	"fmt"
	"log"
	"testing"
)

func TestNewFileDB(t *testing.T) {
	bucketName := "your-gcs-bucket-name" // Replace with your actual GCS bucket name
	fileDB, err := NewFileDB(bucketName)
	if err != nil {
		log.Fatal(err)
	}

	// Create the initial file descriptor
	if err := fileDB.createFileDescriptor(); err != nil {
		log.Fatal(err)
	}

	// Start a goroutine to check for midnight and perform the necessary actions
	go fileDB.checkMidnight(nil)

	// Simulate client writes
	for i := 1; i <= 10; i++ {
		filename := fmt.Sprintf("log%d.txt", i)
		fileDB.writeToDescriptor(filename, fmt.Sprintf("Client write %d\n", i))
	}

	// Close all file descriptors
	fileDB.closeFileDescriptors()
}
