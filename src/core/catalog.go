package core

type Catalog map[string][]*File

func NewCatalog(d DeviceList, f *FileList) Catalog {
	var dSize uint64
	t := make(Catalog)
	dNum := 0
	for fx, fy := range *f {
		if (dSize + fy.Size) <= d[dNum].SizeBytes {
			dSize += fy.Size
		} else {
			dNum += 1
		}
		t[d[dNum].Name] = append(t[d[dNum].Name], &(*f)[fx])
	}
	return t
}
