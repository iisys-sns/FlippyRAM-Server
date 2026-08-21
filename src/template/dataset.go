package template

type Dataset struct {
	Hash string
}

func parseDataset(hash string) *Dataset{
	var dataset Dataset
	dataset.Hash = hash
	return &dataset
}
