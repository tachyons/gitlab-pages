package httperrors

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// creates a new implementation of http.ResponseWriter that allows the
// casting of values in order to aid testing efforts.
type testResponseWriter struct {
	status  int
	content string
	http.ResponseWriter
}

func newTestResponseWriter(w http.ResponseWriter) *testResponseWriter {
	return &testResponseWriter{0, "", w}
}

func (w *testResponseWriter) Status() int {
	return w.status
}

func (w *testResponseWriter) Content() string {
	return w.content
}

func (w *testResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *testResponseWriter) Write(data []byte) (int, error) {
	w.content = string(data)
	return w.ResponseWriter.Write(data)
}

func (w *testResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

var (
	testingContent = content{
		http.StatusNotFound,
		"Title",
		"533",
		"Header test",
		"subheader text",
	}
)

func TestGenerateemailHTML(t *testing.T) {
	actual := generateErrorHTML(testingContent)
	assert.Contains(t, actual, testingContent.title)
	assert.Contains(t, actual, testingContent.statusString)
	assert.Contains(t, actual, testingContent.header)
	assert.Contains(t, actual, testingContent.subHeader)
}

func TestServeErrorPage(t *testing.T) {
	w := newTestResponseWriter(httptest.NewRecorder())
	serveErrorPage(w, testingContent)
	assert.Equal(t, w.Header().Get("Content-Type"), "text/html; charset=utf-8")
	assert.Equal(t, w.Header().Get("X-Content-Type-Options"), "nosniff")
	assert.Equal(t, w.Status(), testingContent.status)
}

func TestServe404(t *testing.T) {
	w := newTestResponseWriter(httptest.NewRecorder())
	Serve404(w)
	assert.Equal(t, w.Header().Get("Content-Type"), "text/html; charset=utf-8")
	assert.Equal(t, w.Header().Get("X-Content-Type-Options"), "nosniff")
	assert.Equal(t, w.Status(), content404.status)
	assert.Contains(t, w.Content(), content404.title)
	assert.Contains(t, w.Content(), content404.statusString)
	assert.Contains(t, w.Content(), content404.header)
	assert.Contains(t, w.Content(), content404.subHeader)
}

func TestServe500(t *testing.T) {
	w := newTestResponseWriter(httptest.NewRecorder())
	Serve500(w)
	assert.Equal(t, w.Header().Get("Content-Type"), "text/html; charset=utf-8")
	assert.Equal(t, w.Header().Get("X-Content-Type-Options"), "nosniff")
	assert.Equal(t, w.Status(), content500.status)
	assert.Contains(t, w.Content(), content500.title)
	assert.Contains(t, w.Content(), content500.statusString)
	assert.Contains(t, w.Content(), content500.header)
	assert.Contains(t, w.Content(), content500.subHeader)
}

func TestServe502(t *testing.T) {
	w := newTestResponseWriter(httptest.NewRecorder())
	Serve502(w)
	assert.Equal(t, w.Header().Get("Content-Type"), "text/html; charset=utf-8")
	assert.Equal(t, w.Header().Get("X-Content-Type-Options"), "nosniff")
	assert.Equal(t, w.Status(), content502.status)
	assert.Contains(t, w.Content(), content502.title)
	assert.Contains(t, w.Content(), content502.statusString)
	assert.Contains(t, w.Content(), content502.header)
	assert.Contains(t, w.Content(), content502.subHeader)
}
