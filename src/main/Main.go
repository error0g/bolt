package main

import (
	"bolt"
	"fmt"
	"log"
)

func main() {
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
	bucket, err := tx.CreateBucket([]byte("MyBucket"))
	if err != nil {
		fmt.Println(err)
	}
	bucket.Put([]byte("foo"), []byte("bar"))

	// Commit the transaction and check for error.
	if err := tx.Commit(); err != nil {
		fmt.Println(err)
	}

}
