---
title: 阅读BoltDB源码
abbrlink: aeeff7df
date: 2022-06-19 14:56:39
tags:
---
# 阅读建议
文章由浅入深，出现新概念会用一两句话介绍，而不用继续陷入分支细节。
阅读时最好具备(Go、Java、C++)其中一门语言。些少的数据库知识，如果你还在处于想基础的SQL怎么写建议先学基础，再回来阅读。

BoltDB整体很简单，没有SQL,没有分布式等等，只是一个纯粹的带简单事务的数据库。

# BoltDB 的介绍

Bolt 一个 Go 语言的嵌入式 KV 数据库，支持多读一写的事务。它的底层是由B+树组织，这颗 B+ 树跟教科书的不一样， 它没有固定的数量的节点。
事务都在运行时内存存储，等待事务 Commit 时相关数据才进行落盘，如果事务提交阶段，在写入磁盘时突然宕机，会使用"备份"进行恢复上一个已写入磁盘的事务状态。Bolt 有 namespance 概念，相当于 MySQL 的表，叫做 Bucket（桶），Bucket可以嵌套 Bucket。
Bolt核心代码很少 4k 左右，结构分明。下方是一段简单往数据库插入一条 **key**:foo **value**:bar 数据的操作。

```Go
//读取数据库文件
db, _ := bolt.Open("my.db", 0600, nil)
defer db.Close()
//创建一个写事务
tx, _ := db.Begin(true)
//在事务里创建Bucket
bucket := tx.Bucket([]byte("MyBucket"))
bucket.Put([]byte("foo"), []byte("bar"))
//事务提交
tx.Commit()
```

# BoltDB 的存储结构

## 磁盘存储结构

BoltDB 存储文件结构是内部是按页管理，为了避免浪费磁盘读取至内存 IO 切换。在代码体现为`Page.go`这个源码文件。
如上方代码`bolt.Open("my.db", 0600, nil)`，如果磁盘未创建即创一个 my.db 文件，第一次创建时会默认创建四个页，Meta0、Meta1、FreeList、LeafPage。
Meta 页为元数据页，FreeList 页用于管理事务释放后的空闲页面，LeafPage 为数据页面。等达到一定数据时会出现 BranchPage，作为索引页，它跟LeafPage是一种结构只不过用一个标记去区分。



下方图为数据库初始化时磁盘文件布局，[db.go#init()](https://github.com/boltdb/bolt/blob/fd01fc79c553a8e99d512a07e8e0c63d4a3ccfc5/db.go#L343-L387)
可以到这个方法看具体代码。
![db_file_from](./img/db_file.png)</br>
Meta0和Meta1是一个结构相当于备份，在事务Commit时，写入磁盘中崩溃可以使用其中一个进行恢复。

```go
func (db *DB) meta() *meta {
    metaA := db.meta0
    metaB := db.meta1
//比较事务Id，事务Id是递增+1的那个最大的Id就为最新事务
 if db.meta1.txid > db.meta0.txid {
    metaA = db.meta1
    metaB = db.meta0
 }
//最新的事务不一定可用，所以如果最新事务不可用就使用“备份”的。
if err := metaA.validate(); err == nil {
    return metaA
 } else if err := metaB.validate(); err == nil {
    return metaB
 }
}
```

## 内存存储结构

# 资源管理

## 内存管理

## 磁盘页管理

# B+树的操作

## 查找元素

## 添加元素

## 删除元素

## 树平衡

### 分裂

### 合并

# 事务

## 读事务

## 写事务

## 事务回滚

# 参考文章

[boltdb 源码分析-我叫尤加利](https://youjiali1995.github.io/storage/boltdb/) <br/>
[自底向上分析boltdb](https://github.com/jaydenwen123/boltdb_book)


