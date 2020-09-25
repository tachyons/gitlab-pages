package main

import (
	"mime"

	"gitlab.com/gitlab-org/labkit/log"
)

var extraMIMETypes = map[string]string{
	".avif": "image/avif",
}

func addExtraMIMETypes() {
	for ext, mimeType := range extraMIMETypes {
		if err := mime.AddExtensionType(ext, mimeType); err != nil {
			log.WithError(err).Errorf("failed to add extension: %q with MIME type: %q", ext, mimeType)
		}
	}
}
