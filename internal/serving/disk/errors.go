package disk

type locationDirectoryError struct {
	FullPath     string
	RelativePath string
}

type locationFileNoExtensionError struct {
	FullPath string
}

func (l *locationDirectoryError) Error() string {
	return "location error accessing directory where file expected"
}

func (l *locationFileNoExtensionError) Error() string {
	return "error accessing a path without an extension"
}
