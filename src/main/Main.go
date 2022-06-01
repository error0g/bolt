package main

import (
	"bolt"
	"fmt"
	"log"
)

func main() {
	//seek过程中把pageId转换为Node或者Page
	db, err := bolt.Open("my.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Start a writable transaction.
	tx, err := db.Begin(true)
	if err != nil {
		fmt.Println(err)
	}
	defer tx.Rollback()

	// Use the transaction...
	bucket := tx.Bucket([]byte("MyBucket"))
	bucket.Put([]byte("5"), []byte("5"))
	// Commit the transaction and check for error.
	if err := tx.Commit(); err != nil {
		fmt.Println(err)
	}

}
