package mimedb

import (
	"mime"
)

//go:generate go run cmd/generate/main.go

func LoadTypes() error {
	// mimeTypeToExts is declared in the generated_mime_types.go file
	for mimeType, extensions := range mimeTypeToExts {
		for _, extension := range extensions {
			if err := mime.AddExtensionType("."+extension, mimeType); err != nil {
				return err
			}
		}
	}

	return nil
}
