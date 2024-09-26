package demo

type SearchRequest struct {
	Author   string
	Classes  []string //类别，命中一个即可
	Keywords []string //关键词，必须全部命中
	ViewFrom int      //视频播放量下限
	ViewTo   int      //视频播放量上限
}
