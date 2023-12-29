package reverseindex

type IReverseIndexer interface {
	Add(DocId uint32, FeatureBits uint32, Keywords []string) //添加一个doc
	Delete(DocId uint32, Keyword string)                     //从key上删除对应的doc
}
