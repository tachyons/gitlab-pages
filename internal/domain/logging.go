package domain

import (
	"net/http"

	"gitlab.com/gitlab-org/labkit/log"
)

func LogFields(r *http.Request) log.Fields {
	logFields := log.Fields{
		"pages_https": r.URL.Scheme == "https",
		"pages_host":  GetHost(r),
	}

	d := FromRequest(r)
	if d == nil {
		return logFields
	}

	lp, err := d.GetLookupPath(r)
	if err != nil {
		logFields["error"] = err.Error()
		return logFields
	}

	logFields["pages_project_serving_type"] = lp.ServingType
	logFields["pages_project_prefix"] = lp.Prefix
	logFields["pages_project_id"] = lp.ProjectID

	return logFields
}
