package core

// FileIndex is a list of file data retrieved from the backup paths.
type FileIndex []*File

// Add a file to the index
func (f *FileIndex) Add(file *File) {
	*f = append(*f, file)
}

// TotalSize returns the byte sum of all file types.
func (f *FileIndex) TotalSize() uint64 {
	var total uint64
	for _, file := range *f {
		total += file.Size
	}
	return total
}

// TotalSizeFiles returns the byte sum of all the files only.
func (f *FileIndex) TotalSizeFiles() uint64 {
	var total uint64
	for _, file := range *f {
		if file.FileType == FILE {
			total += file.Size
		}
	}
	return total
}

// FileByName return a pointer to the named file.
func (f *FileIndex) FileByName(name string) (*File, error) {
	for xx, xy := range *f {
		if xy.Name == name {
			return (*f)[xx], nil
		}
	}
	return nil, new(FileNotFoundError)
}

// DeviceFiles returns all of the destination file objects that are to be copied to the named device.
func (f *FileIndex) DeviceFiles(deviceName string) []*DestFile {
	var files []*DestFile
	for _, file := range *f {
		for _, df := range file.DestFiles {
			if df.DeviceName == deviceName {
				files = append(files, df)
			}
		}
	}
	return files
}
