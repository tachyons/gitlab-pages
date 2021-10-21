package domain

import (
	"net/http"

	"gitlab.com/gitlab-org/labkit/log"
)

func (d *Domain) LogFields(r *http.Request) log.Fields {
	if d == nil {
		return log.Fields{}
	}

	lp, err := d.GetLookupPath(r)
	if err != nil {
		return log.Fields{"error": err.Error()}
	}

	return log.Fields{
		"pages_project_serving_type": lp.ServingType,
		"pages_project_prefix":       lp.Prefix,
		"pages_project_id":           lp.ProjectID,
	}
}
