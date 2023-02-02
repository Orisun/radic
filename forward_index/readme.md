# 正排索引
## 正排结构
正排索引即DocId-->DocInfo的索引， DocInfo通常很大，所以正排索引要存在磁盘中。<br>
```gotemplate
type DocInfo struct {
    DocId uint32
    CompositeFeature uint32 //32个bit分别对应32个bool属性
    Keyword []*Keyword //倒排索引的key
    RankScore float32
    Entry []byte        //正排详情，自行做序列化，比如用protobuf
}
```
Keyword 、DocId、 CompositeFeature用于构建倒排。 RankScore放内存中，遍历倒排链时需要根据RankScore对Doc进行排序。磁盘开销主要用在Entry上。
## 存储引擎
顺序读写磁盘快于随机读写主存。
![avatar](../img/ram.png)
### B+树
![avatar](../img/btree.png)
- B即Balance，对于m叉树每个节点上最多有m个数据，最少有m/2个数据（根节点除外）。
- 叶节点上存储了所有数据，把叶节点链接起来可以顺序遍历所有数据。
- 每个节点设计成内存页的整倍数。MySQL的m=1200，树的前两层放在内存中。
### 跳表
![avatar](../img/skiplist.png)
- 跳表由多条并联的链表组成。查找时当前元素小于target就观察链表的下一个元素，大于target就遍历下一层的链表。
- 沿着最底层的链表可以顺序遍历所有数据。
- 如果倒排链用跳表，将大大提高2条倒排链取交集的速度。
### LSM树
B树每插入一条数据都要写磁盘，而Log-Structured Merge-tree先将大量要写的数据缓存到内存（small tree），积攒到一定量后再批量写入磁盘（big tree），充分利用“顺序写”和“磁盘块”的优势。
![avatar](../img/bigtree.png)
![avatar](../img/smalltree.png)
- tree可以是任意的有序的数据结构。
- small tree长到一定大小时Merge到big tree里面去。
### Succinct-Tire
![avatar](../img/trie.png)
- Tire本身实现了对数据的压缩，而作为树它还可以进一步压缩。
- 水平遍历对节点编号。
- 有几孩子编码中就有几个1。
- 19个bits就可以表示整棵树。
![avatar](../img/succinct.png)
- rank1(x)—在range[0,x]里面1的个数。
- select1(y)—第y个1所在的位置。
- 根据rank和select函数可以快速定位到任意节点的父节点和子节点，从而实现对树的遍历。
### key/value存储引擎
![avatar](../img/kvengine.png)
- 上面这个表格是官方文档，不要太当真。生产环境实测，bolt比rocksdb慢27%，badger和rocksdb速度相当，但badger对CPU和内存的浪费是巨大的，远没有rocksdb节俭。所以数据量小时建议用badger(安装go rocksdb太麻烦了，参见<a href="install_rocksdb.md" alt="install_rocksdb.md">我的安装经历</a>)，数据量大时建议用rocksdb。  
- 当随机读是主要操作时，Rocksdb的参数调优：
  - memtable选用HashSkipList
  - SST采用PlainTable
  - 采用Level Compaction
- 分成多个Shard，提高并行能力。
