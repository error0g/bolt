---
title: 阅读BoltDB源码
abbrlink: aeeff7df
date: 2022-06-19 14:56:39
tags:
---

# 阅读建议

文章由浅入深，出现新概念会用一两句话介绍，而不用继续陷入分支细节。
阅读时最好具备(Go、Java、C++)其中一门语言。一些少许的数据库知识，如果你还在处于不知道基础的 SQL 怎么写的阶段建议先学基础，再回来继续阅读。

BoltDB 整体很简单，没有 SQL ，没有分布式，等等，只是一个纯粹的带简单事务的数据库。

//TODO 描述文章的路径


# BoltDB 的介绍

**原介绍**  项目地址：[github](https://github.com/boltdb/bolt)

Bolt is a pure Go key/value store inspired by Howard Chu's LMDB project. The goal of the project is to provide a simple, fast, and reliable database for projects that don't require a full database server such as Postgres or MySQL.

Since Bolt is meant to be used as such a low-level piece of functionality, simplicity is key. The API will be small and only focus on getting values and setting values. That's it.

Bolt 一个 Go 语言的嵌入式 KV 数据库，支持多读一写的事务。它的底层是由B+树组织，这颗 B+ 树跟教科书的不一样， 它没有固定的数量的节点，叶子结点没有关联。
事务都在运行时内存存储，等待事务 Commit 时相关数据才进行落盘，如果事务提交阶段，在写入磁盘时突然宕机，会使用"备份"进行恢复上一个已写入磁盘的事务状态。Bolt 有 namespance 概念，相当于 MySQL 的表，叫做
Bucket（桶），Bucket 可以嵌套 子Bucket。
Bolt 核心代码很少 4k 左右，结构分明。下方是一段往数据库插入一条 **key**:foo **value**:bar 数据的操作。

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

## 磁盘存储布局
BoltDB 磁盘文件结构是内部是按页管理，为了避免浪费磁盘读取至内存 IO 切换，一个页面大小为 4096=2^12，跟磁盘扇区大小一致，一个磁盘扇区是磁盘读写最小单位。在代码体现为`Page.go`这个源码文件。
如上方代码`bolt.Open("my.db", 0600, nil)`，如果磁盘未创建即创一个 my.db 文件，第一次创建时会默认创建四个页，Meta0Page、Meta1Page、FreeList、LeafPage。
Meta 页为元数据页用于描述数据库文件中其他页在哪，FreeList 页用于管理事务释放后的空闲页面，LeafPage 为数据页面。等达到一定数据时会出现 BranchPage，作为索引页，它跟LeafPage是一种结构只不过用一个标记去区分。

下方图为数据库初始化时磁盘文件布局，[db.go#init()](https://github.com/boltdb/bolt/blob/fd01fc79c553a8e99d512a07e8e0c63d4a3ccfc5/db.go#L343-L387)
可以到这个方法看具体代码。
![db_file_from](./img/db_file.png)</br>

## 基础页面Page
所有的page都是基于这个结构构建，page这个数据结构分为两个部分，(id...)和ptr，前部分相当于描述这个页，后部分是页的具体内容。其他字段看注释很好理解，overflow 字段是因为一个树结点可能占用几个页面，它的原理是记录溢出页面数量，通过递增页面Id可访问到。在commit的时候才会分配这些页面。

```go 
type page struct {
   //页id 
   id       pgid
   //区分页类型
   flags    uint16
   //BranchPage和LeafPage数据元素数量
   count    uint16 
   //溢出连续页面数量
   overflow uint32
   //具体数据
   ptr      uintptr
}
```
ptr无符号指针，指向内存。可以把内存当作大数组，而ptr存放的就是数组的index。Index+Size可以获取整个页内容。

![db_file_from](./img/page.png)</br>

page转换meta页样例。
```go
func (p *page) meta() *meta {
    return (*meta)(unsafe.Pointer(&p.ptr))  //&p.ptr=首地址+meta指针需要的内存大小
}
```
![page_meta](./img/page_meta.png)</br>
## Meta 页
```go
//db.go
type meta struct {
    magic    uint32 
    version  uint32 
    pageSize uint32 
    flags    uint32 
    root     bucket //树的根节点
    freelist pgid   
    pgid     pgid  //当前最大页面id范围
    txid     txid  //事务id
    checksum uint64 
}
```
把 meta 分成三个部分 (magic、version,flags)、(root,freelist)、(txid,checksum)。第一部分用于文件是否正确，第二部分用于逻辑关联其他页（可以看到下方图），第三部分用于校验meta事务是否可用。
Meta0 和 Meta1 是一样的结构，相当于备份，在事务Commit时，写入磁盘中崩溃可以使用其中一个进行恢复。

可以看到 root 字段是 bucket 类型，其实bucket里面包装了 pgid。从上文知道一个页是4096字节，那么访问数据库文件的 0x0000 至 0x1000 字节就是第一个 meta 页面，（4096+4096）是第二个页面，那么得到 meta 页面根据其他 pgid*4096 即可访问到其他页面数据。
![meta](./img/meta.png)

有两个 meta 页很必要，事务有一个特性就是数据一致性，如果在一个事务中提交写入磁盘阶段，一部分数据写入了磁盘但是突然发生崩溃，导致另一部分相关的数据没有写入磁盘就会产生脏数据。数据库每次写入只使用一个meta页面，开启事务时候会复制一个meta页给事务，写入磁盘的时候是写到另外一个meta页面，这样不会干扰到之前的正常meta页面。在启动数据库时会校验 meta 是否完整，选择完整的 meta 进行恢复，可以看到下方代码。
```go
//db.go
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
## Freelist 页
数据库文件内部是按页的单位管理，这就涉及到如何分配页和回收页（要释放旧的页，旧的页面是从数据复制到新事务）。事务提交的阶段会把树结点分写入到一个页或多个页（页大小4096字节），根据磁盘特性顺序写比随机写要快，所以在分配页面时如果有溢出页会开辟连续的页。
![freelist](./img/freelist.png)
可以看到上方图事务修改后 pgid:3 的页面变成了脏页，会分配新的页面 pgid:4，从新刷入磁盘。这样可以达到事务隔离，旧的事务只看到 pgid:3 页面,而无法看到新事务的新页面（因为页面会由meta关联）。
```go
//freelist.go
type freelist struct {
    ids []pgid  //空闲页 id 列表
    pending map[txid][]pgid //每个事务将要释放的旧页面id
    cache   map[pgid]bool  //找所有空闲和待处理的页面id
}
```
分配页面都是等待事务commit时调用 freelist.go#allocate() 方法才去分配，如果文件大小不够会开辟空间重新映射。 页面释放到pending后没有真正回收到ids，需要等待下一个事务开启才会把上一个事务的旧页释放到空闲页。
## 内存存储结构

# 资源管理

## 内存管理

## 磁盘页管理
```go
//freelist.go
func (f *freelist) allocate(n int) pgid {
	if len(f.ids) == 0 {
		return 0
	}
	var initial, previd pgid
	for i, id := range f.ids {
		if id <= 1 {
		    panic(fmt.Sprintf("invalid page allocation: %d", id))
		}

		if previd == 0 || id-previd != 1 {
			initial = id
		}
		
		if (id-initial)+1 == pgid(n) {
			if (i + 1) == n {
				f.ids = f.ids[i+1:]
			} else {
				copy(f.ids[i-n+1:], f.ids[i+1:])
				f.ids = f.ids[:len(f.ids)-n]
			}
			for i := pgid(0); i < pgid(n); i++ {
				delete(f.cache, initial+i)
			}
			return initial
		}
		previd = id
	}
	return 0
}
```
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
pending
# 参考文章

[boltdb 源码分析-我叫尤加利](https://youjiali1995.github.io/storage/boltdb/) <br/>
[自底向上分析boltdb](https://www.bookstack.cn/read/jaydenwen123-boltdb_book/00fe39712cec954e.md)


