package main

import (
	"bolt"
	"fmt"
	"log"
)

//TODO node.read node.write
func main() {
	//读取数据库文件
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

	// 使用读事务创建一个bucket
	bucket := tx.Bucket([]byte("MyBucket"))

	////插入一个数据
	//bucket.Put([]byte("foo"), []byte("bar"))
	//bucket.Put([]byte("fo1"), []byte("bar"))
	//提交事务
	// Commit the transaction and check for error.
	if err := tx.Commit(); err != nil {
		fmt.Println(err)
	}
}
