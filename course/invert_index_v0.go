package course

type Doc struct {
	Id       int
	Keywords []string
}

func BuildInvertIndex(docs []*Doc) map[string][]int {
	index := make(map[string][]int, 100)
	for _, doc := range docs {
		for _, keyword := range doc.Keywords {
			index[keyword] = append(index[keyword], doc.Id)
		}
	}
	return index
}
