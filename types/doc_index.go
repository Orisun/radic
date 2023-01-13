// Date         : 2018/7/28
// Author       : zhangchaoyang
// Description  : 
package types

import (
	"strings"
)

const (
	nil_keyword = "nil"
)

//DocInfo 往索引里添加文档时需要准确好这些字段，这也是正排索引上需要存储的信息
type DocInfo struct {
	DocId            uint32
	CompositeFeature uint32     //32个bit分别对应32个bool属性
	Keyword          []*Keyword //倒排索引的key
	RankScore        float32
	Entry            []byte     //正排详情，自行做序列化
}

func (self *DocInfo) EntryEqual(other *DocInfo) bool {
	if self.Entry == nil && other.Entry == nil {
		return true
	}
	if len(self.Entry) != len(other.Entry) {
		return false
	}
	for i, ele := range self.Entry {
		if ele != other.Entry[i] {
			return false
		}
	}
	return true
}

func (self *Keyword) ToString() string {
	if self == nil {
		return nil_keyword
	}
	return self.Field + "\001" + self.Word
}

func NewKeyword(ser string) *Keyword {
	if ser == nil_keyword {
		return nil
	}
	arr := strings.Split(ser, "\001")
	if len(arr) == 2 {
		return &Keyword{Field: arr[0], Word: arr[1]}
	} else {
		return nil
	}
}

//ScoredDoc 参与排序的文档
type ScoredDoc struct {
	DocId     uint32
	RankScore float32
}

//ScoredDocs 给文档排序
type ScoredDocs []ScoredDoc

func (docs ScoredDocs) Len() int {
	return len(docs)
}

func (docs ScoredDocs) Swap(i, j int) {
	docs[i], docs[j] = docs[j], docs[i]
}

func (docs ScoredDocs) Less(i, j int) bool {
	return docs[i].Less(docs[j])
}

func (doc ScoredDoc) Less(other ScoredDoc) bool {
	//得分高的排前面
	if doc.RankScore > other.RankScore {
		return true
	} else {
		return false
	}
}

//TopK 把最大的前K个元素放在前K个位置
func (docs ScoredDocs) TopK(k int) {
	docs.topK(k, 0, docs.Len())
}

func (docs ScoredDocs) topK(k int, begin int, end int) {
	if k >= end-begin {
		return
	}
	if begin >= end {
		return
	}
	pivot := docs[begin]
	low := begin
	high := end - 1
	for ; low < high; {
		for ; low < high && !docs[high].Less(pivot); { //注意不能用pivot.Less(docs[high])，否则当有重复元素时会发生死循环
			high--
		}
		docs[low] = docs[high]
		for ; low < high && !pivot.Less(docs[low]); {
			low++
		}
		docs[high] = docs[low]
	}
	docs[low] = pivot

	if low+1 == k {
		return
	} else if low+1 < k {
		docs.topK(k-low-1, low+1, end)
	} else {
		docs.topK(k, begin, low)
	}
}
