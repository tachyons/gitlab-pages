package main

import (
	"fmt"
	"mime"

	"gitlab.com/gitlab-org/labkit/log"
)

var extraMIMETypes = map[string]string{
	".avif": "image/avif",
}

func addExtraMIMETypes() {
	fmt.Printf("calling addExtraMIMETypes:  %+v\n", extraMIMETypes)
	for ext, mimeType := range extraMIMETypes {
		if err := mime.AddExtensionType(ext, mimeType); err != nil {
			fmt.Printf("failed %q - %+v\n", mimeType, err)
			log.WithError(err).Errorf("failed to add extension: %q with MIME type: %q", ext, mimeType)
		} else {
			fmt.Printf("loaded %q successfully\n", mimeType)
		}
	}
}
