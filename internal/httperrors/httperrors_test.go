package httperrors

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
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
	require.Contains(t, actual, testingContent.title)
	require.Contains(t, actual, testingContent.statusString)
	require.Contains(t, actual, testingContent.header)
	require.Contains(t, actual, testingContent.subHeader)
}

func TestServeErrorPage(t *testing.T) {
	w := newTestResponseWriter(httptest.NewRecorder())
	serveErrorPage(w, testingContent)
	require.Equal(t, w.Header().Get("Content-Type"), "text/html; charset=utf-8")
	require.Equal(t, w.Header().Get("X-Content-Type-Options"), "nosniff")
	require.Equal(t, w.Status(), testingContent.status)
}

func TestServe401(t *testing.T) {
	w := newTestResponseWriter(httptest.NewRecorder())
	Serve401(w)
	require.Equal(t, w.Header().Get("Content-Type"), "text/html; charset=utf-8")
	require.Equal(t, w.Header().Get("X-Content-Type-Options"), "nosniff")
	require.Equal(t, w.Status(), content401.status)
	require.Contains(t, w.Content(), content401.title)
	require.Contains(t, w.Content(), content401.statusString)
	require.Contains(t, w.Content(), content401.header)
	require.Contains(t, w.Content(), content401.subHeader)
}

func TestServe404(t *testing.T) {
	w := newTestResponseWriter(httptest.NewRecorder())
	Serve404(w)
	require.Equal(t, w.Header().Get("Content-Type"), "text/html; charset=utf-8")
	require.Equal(t, w.Header().Get("X-Content-Type-Options"), "nosniff")
	require.Equal(t, w.Status(), content404.status)
	require.Contains(t, w.Content(), content404.title)
	require.Contains(t, w.Content(), content404.statusString)
	require.Contains(t, w.Content(), content404.header)
	require.Contains(t, w.Content(), content404.subHeader)
}

func TestServe414(t *testing.T) {
	w := newTestResponseWriter(httptest.NewRecorder())
	Serve414(w)
	require.Equal(t, w.Header().Get("Content-Type"), "text/html; charset=utf-8")
	require.Equal(t, w.Header().Get("X-Content-Type-Options"), "nosniff")
	require.Equal(t, w.Status(), content414.status)
	require.Contains(t, w.Content(), content414.title)
	require.Contains(t, w.Content(), content414.statusString)
	require.Contains(t, w.Content(), content414.header)
	require.Contains(t, w.Content(), content414.subHeader)
}

func TestServe500(t *testing.T) {
	w := newTestResponseWriter(httptest.NewRecorder())
	Serve500(w)
	require.Equal(t, w.Header().Get("Content-Type"), "text/html; charset=utf-8")
	require.Equal(t, w.Header().Get("X-Content-Type-Options"), "nosniff")
	require.Equal(t, w.Status(), content500.status)
	require.Contains(t, w.Content(), content500.title)
	require.Contains(t, w.Content(), content500.statusString)
	require.Contains(t, w.Content(), content500.header)
	require.Contains(t, w.Content(), content500.subHeader)
}

func TestServe502(t *testing.T) {
	w := newTestResponseWriter(httptest.NewRecorder())
	Serve502(w)
	require.Equal(t, w.Header().Get("Content-Type"), "text/html; charset=utf-8")
	require.Equal(t, w.Header().Get("X-Content-Type-Options"), "nosniff")
	require.Equal(t, w.Status(), content502.status)
	require.Contains(t, w.Content(), content502.title)
	require.Contains(t, w.Content(), content502.statusString)
	require.Contains(t, w.Content(), content502.header)
	require.Contains(t, w.Content(), content502.subHeader)
}
