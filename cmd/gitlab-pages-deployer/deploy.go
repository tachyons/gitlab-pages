package main

import (
	workers "github.com/jrallison/go-workers"
)

func deployJob(message *workers.Msg, projectID int64, projectPath string, jobID int64, config map[string]interface{}) {
	saveConfig(projectID, projectPath, config)
}

func configJob(message *workers.Msg, projectID int64, projectPath string, config map[string]interface{}) {
	saveConfig(projectID, projectPath, config)
}
