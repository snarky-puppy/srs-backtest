package pp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"cloud.google.com/go/storage"
)

type FileDB struct {
	ctx                context.Context
	client             *storage.Client
	bucketName         string
	fileDescriptors    map[string]*storage.Writer
	currentFileLabel   string
	fileDescriptorsMux sync.Mutex
	DoneCh             chan struct{}
}

func NewFileDB(ctx context.Context, bucketName string) (*FileDB, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	// check client access
	_, err = client.Bucket(bucketName).Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("bucket %s doesn't exist", bucketName)
	}

	rv := &FileDB{
		ctx:              ctx,
		client:           client,
		bucketName:       bucketName,
		fileDescriptors:  make(map[string]*storage.Writer),
		currentFileLabel: time.Now().Format("2006-01-02"),
		DoneCh:           make(chan struct{}),
	}

	go rv.sync(ctx)

	return rv, nil
}

func (db *FileDB) createFileDescriptor(filename string) (*storage.Writer, error) {
	for i := 0; ; i++ {
		uniqueFilename := fmt.Sprintf("%s_%s_%d", filename, db.currentFileLabel, i)

		object := db.client.Bucket(db.bucketName).Object(uniqueFilename)

		attrs, err := object.Attrs(db.ctx)
		if err != nil {
			if err == storage.ErrObjectNotExist {
				writer := object.NewWriter(context.Background()) // don't use global ctx here, cancel will close the writer without writing
				db.fileDescriptors[filename] = writer
				return writer, nil
			}
			return nil, err
		}

		if attrs.Size == 0 {
			writer := object.NewWriter(db.ctx)
			db.fileDescriptors[filename] = writer
			return writer, nil
		}
	}
}

func (db *FileDB) sync(ctx context.Context) {
	now := time.Now()
	nextFifteen := now.Truncate(15 * time.Minute).Add(15 * time.Minute).Sub(now)

	log.Printf("Next sync is %s", nextFifteen)

	for {
		select {
		case <-ctx.Done():
			db.closeFileDescriptors()
			close(db.DoneCh)
			return
		case <-time.After(nextFifteen):
			db.fileDescriptorsMux.Lock()

			// Flush and close the current file descriptors
			db.closeFileDescriptors()

			// Create a new file descriptor for the new day
			db.currentFileLabel = time.Now().Format("2006-01-02")

			db.fileDescriptorsMux.Unlock()

			nextFifteen = 15 * time.Minute
			log.Printf("Next sync is %s", nextFifteen)
		}
	}
}

func (db *FileDB) closeFileDescriptors() {
	for _, writer := range db.fileDescriptors {
		err := writer.Close()
		if err != nil {
			log.Printf("Error closing file descriptor: %v", err)
		}
	}
	db.fileDescriptors = make(map[string]*storage.Writer)
}

func (db *FileDB) Write(name string, data interface{}) {
	fd, ok := db.fileDescriptors[name]
	if !ok {
		db.fileDescriptorsMux.Lock()
		var err error
		fd, err = db.createFileDescriptor(name)
		db.fileDescriptorsMux.Unlock()
		if err != nil {
			log.Printf("error creating file descriptor: %v", err)
		}
	}
	err := json.NewEncoder(fd).Encode(data)
	if err != nil {
		log.Printf("error encoding data: %v", err)
	}
}
