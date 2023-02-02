# 倒排索引
## 倒排链的基本结构
![avatar](../img/invert_index.png)
倒排链上的每个元素是uint64，由2个uint32拼接而成，第一个uint32是doc id，第二个uint32是CompositeFeature，CompositeFeature可以表示doc的32个bool属性。<br>
搜索"工业机器学习"，即求第一个和第四个倒排链的交集，返回doc id的集合{1,5,2}。<br>
整个倒排索引是由多个map构成的。没有采用ConcurrentHashMap，而是用原生的Map，读写时对整个map加锁，为减少并发等待，分了1000个Map。
## bits过滤
搜索经常带筛选条件，比如只搜价格在50-100元之间的英文图书。如果倒排链上只存doc id就无法完成筛选功能，如果倒排链上存储doc详情则内存容不下，折中的办法是倒排链上存储doc的bool属性，这样4个字节就可以存下32个bool
属性。<br>
![avatar](../img/bit1.png)
检索时传入3个int32参数，on表示对应的bit位必须全是1，off表示对应的bit位必须全是0，or表示对应的bit位必须至少有一个是1，即：<br>
```
bits & on == on
bits & off == 0
bits & or > 0
```
![avatar](../img/bit2.png)
32个bit通常会有富余，可以拿出一些bit来表示doc的连续属性。如上图所示，对价格进行了离散化，每一段都是闭区间，"80元"刚好位于2个区间的交接点上，则这2个区间都置为1。
## 超时控制
- Query中有多个关键词时各条链并行遍历
- 单条倒排链较长时并行遍历
- 单条倒排链最多只读前50万个doc
- 倒排链取交集后，取RankScore（RankScore放内存中）最大的前6000个，其余的丢弃，因为接下来读正排索引是非常耗时的
## 性能调优
- Goroutin的量要克制，虽然可以轻松地创建上百万个routine，但真正执行的不超过1万。
- 用slice替代并发容器（channel和ConcurrentHashMap）
  - 并发容器都要加锁
  - Golang中的array、slice、struct都支持并发读写，如果确定不存在race condition可以不加锁
  - map不支持并发读写
- 防止频繁GC
  - 在make map和make slice时全部指定capacity
  - 每次遍历倒排链时都要创建一个int数组以容纳满足bits的结果，此时可以创建一个生产int数组的对象池sync.Pool
