package example

import "fmt"

type File struct {
	Name string
	Size int64
}

func describe(file File) string {
	return fmt.Sprintf("%s (%d bytes)", file.Name, file.Size)
}
