package types

type Keyword struct {
	Field string
	Word  string
}

func (kw Keyword) ToString() string {
	return kw.Field + "\001" + kw.Word
}

type Document struct {
	Id          string     //业务使用的唯一Id，索引上此Id不会重复
	IntId       uint64     //倒排索引上使用的文档id(业务侧不用管这个字段)
	BitsFeature uint64     //每个bit都表示某种特征的取值
	Keywords    []*Keyword //倒排索引的key
	Bytes       []byte     //业务实体序列化之后的结果
}
