package httperrors

import (
	"fmt"
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/errortracking"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
)

type content struct {
	status       int
	title        string
	statusString string
	header       string
	subHeader    string
}

var (
	content401 = content{
		http.StatusUnauthorized,
		"Unauthorized (401)",
		"401",
		"You don't have permission to access the resource.",
		`<p>The resource that you are attempting to access is protected and you don't have the necessary permissions to view it.</p>`,
	}
	content404 = content{
		http.StatusNotFound,
		"The page you're looking for could not be found (404)",
		"404",
		"The page you're looking for could not be found.",
		`<p>The resource that you are attempting to access does not exist or you don't have the necessary permissions to view it.</p>
     <p>Make sure the address is correct and that the page hasn't moved.</p>
     <p>Please contact your GitLab administrator if you think this is a mistake.</p>`,
	}
	content414 = content{
		status:       http.StatusRequestURITooLong,
		title:        "Request URI Too Long (414)",
		statusString: "414",
		header:       "Request URI Too Long.",
		subHeader: `<p>The URI provided was too long for the server to process.</p>
			<p>Try to make the request URI shorter.</p>`,
	}

	content429 = content{
		http.StatusTooManyRequests,
		"Too many requests (429)",
		"429",
		"Too many requests.",
		`<p>The resource that you are attempting to access is being rate limited.</p>`,
	}
	content500 = content{
		http.StatusInternalServerError,
		"Something went wrong (500)",
		"500",
		"Whoops, something went wrong on our end.",
		`<p>Try refreshing the page, or going back and attempting the action again.</p>
     <p>Please contact your GitLab administrator if this problem persists.</p>`,
	}

	content502 = content{
		http.StatusBadGateway,
		"Something went wrong (502)",
		"502",
		"Whoops, something went wrong on our end.",
		`<p>Try refreshing the page, or going back and attempting the action again.</p>
     <p>Please contact your GitLab administrator if this problem persists.</p>`,
	}

	content503 = content{
		http.StatusServiceUnavailable,
		"Service Unavailable (503)",
		"503",
		"Whoops, something went wrong on our end.",
		`<p>Try refreshing the page, or going back and attempting the action again.</p>
     <p>Please contact your GitLab administrator if this problem persists.</p>`,
	}
)

const predefinedErrorPage = `
<!DOCTYPE html>
<html>
<head>
  <meta content="width=device-width, initial-scale=1, maximum-scale=1" name="viewport">
  <title>%v</title>
  <style>
    body {
      color: #666;
      text-align: center;
      font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
      margin: auto;
      font-size: 14px;
    }

    h1 {
      font-size: 56px;
      line-height: 100px;
      font-weight: 400;
      color: #456;
    }

    h2 {
      font-size: 24px;
      color: #666;
      line-height: 1.5em;
    }

    h3 {
      color: #456;
      font-size: 20px;
      font-weight: 400;
      line-height: 28px;
    }

    hr {
      max-width: 800px;
      margin: 18px auto;
      border: 0;
      border-top: 1px solid #EEE;
      border-bottom: 1px solid white;
    }

    img {
      max-width: 40vw;
      display: block;
      margin: 40px auto;
    }

    a {
      line-height: 100px;
      font-weight: 400;
      color: #4A8BEE;
      font-size: 18px;
      text-decoration: none;
    }

    .container {
      margin: auto 20px;
    }

    .go-back {
      display: none;
    }

  </style>
</head>

<body>
  <img src="data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjEwIiBoZWlnaHQ9IjIxMCIgdmlld0JveD0iMCAwIDIxMCAyMTAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+CiAgPHBhdGggZD0iTTEwNS4wNjE0IDIwMy42NTVsMzguNjQtMTE4LjkyMWgtNzcuMjhsMzguNjQgMTE4LjkyMXoiIGZpbGw9IiNlMjQzMjkiLz4KICA8cGF0aCBkPSJNMTA1LjA2MTQgMjAzLjY1NDhsLTM4LjY0LTExOC45MjFoLTU0LjE1M2w5Mi43OTMgMTE4LjkyMXoiIGZpbGw9IiNmYzZkMjYiLz4KICA8cGF0aCBkPSJNMTIuMjY4NSA4NC43MzQxbC0xMS43NDIgMzYuMTM5Yy0xLjA3MSAzLjI5Ni4xMDIgNi45MDcgMi45MDYgOC45NDRsMTAxLjYyOSA3My44MzgtOTIuNzkzLTExOC45MjF6IiBmaWxsPSIjZmNhMzI2Ii8+CiAgPHBhdGggZD0iTTEyLjI2ODUgODQuNzM0Mmg1NC4xNTNsLTIzLjI3My03MS42MjVjLTEuMTk3LTMuNjg2LTYuNDExLTMuNjg1LTcuNjA4IDBsLTIzLjI3MiA3MS42MjV6IiBmaWxsPSIjZTI0MzI5Ii8+CiAgPHBhdGggZD0iTTEwNS4wNjE0IDIwMy42NTQ4bDM4LjY0LTExOC45MjFoNTQuMTUzbC05Mi43OTMgMTE4LjkyMXoiIGZpbGw9IiNmYzZkMjYiLz4KICA8cGF0aCBkPSJNMTk3Ljg1NDQgODQuNzM0MWwxMS43NDIgMzYuMTM5YzEuMDcxIDMuMjk2LS4xMDIgNi45MDctMi45MDYgOC45NDRsLTEwMS42MjkgNzMuODM4IDkyLjc5My0xMTguOTIxeiIgZmlsbD0iI2ZjYTMyNiIvPgogIDxwYXRoIGQ9Ik0xOTcuODU0NCA4NC43MzQyaC01NC4xNTNsMjMuMjczLTcxLjYyNWMxLjE5Ny0zLjY4NiA2LjQxMS0zLjY4NSA3LjYwOCAwbDIzLjI3MiA3MS42MjV6IiBmaWxsPSIjZTI0MzI5Ii8+Cjwvc3ZnPgo="
       alt="GitLab Logo" />
  <h1>
    %v
  </h1>
  <div class="container">
    <h3>%v</h3>
    <hr />
    %v
    <a href="javascript:history.back()" class="js-go-back go-back">Go back</a>
  </div>
  <script>
    (function () {
      var goBack = document.querySelector('.js-go-back');

      if (history.length > 1) {
        goBack.style.display = 'inline';
      }
    })();
  </script>
</body>
</html>
`

func generateErrorHTML(c content) string {
	return fmt.Sprintf(predefinedErrorPage, c.title, c.statusString, c.header, c.subHeader)
}

func serveErrorPage(w http.ResponseWriter, c content) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(c.status)
	fmt.Fprintln(w, generateErrorHTML(c))
}

// Serve401 returns a 401 error response / HTML page to the http.ResponseWriter
func Serve401(w http.ResponseWriter) {
	serveErrorPage(w, content401)
}

// Serve404 returns a 404 error response / HTML page to the http.ResponseWriter
func Serve404(w http.ResponseWriter) {
	serveErrorPage(w, content404)
}

// Serve414 returns a 414 error response / HTML page to the http.ResponseWriter
func Serve414(w http.ResponseWriter) {
	serveErrorPage(w, content414)
}

// Serve429 returns a 429 error response / HTML page to the http.ResponseWriter
func Serve429(w http.ResponseWriter) {
	serveErrorPage(w, content429)
}

// Serve500 returns a 500 error response / HTML page to the http.ResponseWriter
func Serve500(w http.ResponseWriter) {
	serveErrorPage(w, content500)
}

// Serve500WithRequest returns a 500 error response / HTML page to the http.ResponseWriter
func Serve500WithRequest(w http.ResponseWriter, r *http.Request, reason string, err error) {
	logging.LogRequest(r).WithError(err).Error(reason)
	errortracking.CaptureErrWithReqAndStackTrace(err, r)
	serveErrorPage(w, content500)
}

// Serve502 returns a 502 error response / HTML page to the http.ResponseWriter
func Serve502(w http.ResponseWriter) {
	serveErrorPage(w, content502)
}

// Serve503 returns a 503 error response / HTML page to the http.ResponseWriter
func Serve503(w http.ResponseWriter) {
	serveErrorPage(w, content503)
}
