package domain

import "net/http"

type handler struct {
	writer  http.ResponseWriter
	request *http.Request
	project *Project
	subpath string
}

func (h *handler) Writer() http.ResponseWriter {
	return h.writer
}

func (h *handler) Request() *http.Request {
	return h.request
}

func (h *handler) LookupPath() string {
	return h.project.LookupPath
}

func (h *handler) Subpath() string {
	return h.subpath
}

func (h *handler) IsNamespaceProject() bool {
	return h.project.IsNamespaceProject
}

func (h *handler) IsHTTPSOnly() bool {
	return h.project.IsHTTPSOnly
}

func (h *handler) HasAccessControl() bool {
	return h.project.HasAccessControl
}

func (h *handler) ProjectID() uint64 {
	return h.project.ID
}
