package teamprops

type AuthorMetricPair struct {
	Author string
	Value  int
}

type AuthorMetricPairList []AuthorMetricPair

func (p AuthorMetricPairList) Len() int           { return len(p) }
func (p AuthorMetricPairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p AuthorMetricPairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
