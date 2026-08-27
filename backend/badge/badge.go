package badge

import "embed"

//go:embed *.png
var files embed.FS

func Read(name string) ([]byte, error) {
	return files.ReadFile(name)
}
