package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jfbus/httprs"
	"github.com/xanzy/go-gitlab"

	workers "github.com/jrallison/go-workers"
)

func writeFile(fullPath string, r io.Reader, size int64) error {
	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()

	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}

	if n != size {
		return fmt.Errorf("%s: readed only %d instead of %d", fullPath, n, size)
	}
	return nil
}

func extractFile(file *zip.File, dir string) error {
	fullPath := filepath.Join(dir, file.Name)
	println(fullPath, "size=", file.UncompressedSize64)

	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	dirPath := filepath.Dir(fullPath) + "/"
	os.MkdirAll(dirPath, 0750)

	if file.Mode().IsRegular() {
		return writeFile(fullPath, rc, int64(file.UncompressedSize64))
	}

	return nil
}

func extractArtifactsArchive(tempDir string, artifactsURL string) error {
	req, err := http.NewRequest("GET", artifactsURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("PRIVATE-TOKEN", *apiAccessToken)

	resp, err := http.DefaultClient.Do(req)
	panicOnHTTPError(resp, err)

	archiveStream := httprs.NewHttpReadSeeker(resp)
	defer archiveStream.Close()

	reader, err := zip.NewReader(archiveStream, resp.ContentLength)

	for _, file := range reader.File {
		// if !strings.HasPrefix(file.Name, "public/") {
		// 	continue
		// }

		err := extractFile(file, tempDir)
		if err != nil {
			return err
		}
	}

	return nil
}

func extractJob(message *workers.Msg, projectID int, projectPath string, artifactsURL string) error {
	tempDir := filepath.Join(*deployerRoot, "tmp")
	os.MkdirAll(tempDir, 0750)

	targetPath := filepath.Join(*deployerRoot, projectPath)
	os.MkdirAll(targetPath, 0750)

	// Create temp directory where we will store artifacts
	deployTempDir, err := ioutil.TempDir(tempDir, "new_deploy")
	if err != nil {
		return err
	}
	defer os.RemoveAll(deployTempDir)

	// Extract archive to deployTempDir
	err = extractArtifactsArchive(filepath.Join(deployTempDir, "public"), artifactsURL)
	if err != nil {
		return err
	}

	oldTargetDir, err := ioutil.TempDir(targetPath, ".deleted")
	if err != nil {
		return err
	}
	defer os.RemoveAll(oldTargetDir)

	// Shuffle directories doing save "atomic" move
	publicTempDir := filepath.Join(deployTempDir, "public")
	publicTargetDir := filepath.Join(targetPath, "public")
	publicOldTargetDir := filepath.Join(oldTargetDir, "public")

	fi, err := os.Stat(publicTempDir)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("public/ has to be present")
		}
		return err
	}
	if !fi.IsDir() {
		return errors.New("public/ has to be directory")
	}

	// Move old target public to a temporary public
	err = os.Rename(publicTargetDir, publicOldTargetDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Move new public to target public
	err = os.Rename(publicTempDir, publicTargetDir)
	if err != nil {
		// If this fails, try to bring back old target dir
		os.Rename(publicOldTargetDir, publicTargetDir)
		return err
	}

	// Use defer to remove: publicOldTarget and deployTempDir
	return nil
}

func deployJob(message *workers.Msg, projectID int, projectPath string, pipelineID int, jobID int, config map[string]interface{}) {
	pipeline, resp, err := api.Pipelines.GetPipeline(projectID, pipelineID)
	panicOnAPIError(resp, err)

	if pipeline == nil {
		return
	}

	println("Pipeline", projectPath, pipeline.ID, pipeline.Sha, pipeline.Ref)

	_, resp, err = api.Commits.SetCommitStatus(projectID, pipeline.Sha, &gitlab.SetCommitStatusOptions{
		State:       gitlab.Running,
		Ref:         &pipeline.Ref,
		Name:        gitlab.String("pages:deploy"),
		Description: gitlab.String("started deploying"),
	})
	panicOnAPIError(resp, err)

	err = extractJob(message, projectID, projectPath,
		fmt.Sprint(*apiURL, "/projects/", projectID, "/jobs/", jobID, "/artifacts"))

	if err != nil {
		println("Pipeline", pipeline.ID, pipeline.Sha, pipeline.Ref, err.Error())
		_, resp, err = api.Commits.SetCommitStatus(projectID, pipeline.Sha, &gitlab.SetCommitStatusOptions{
			State:       gitlab.Failed,
			Ref:         &pipeline.Ref,
			Name:        gitlab.String("pages:deploy"),
			Description: gitlab.String(err.Error()),
		})
		panicOnAPIError(resp, err)
		return
	}

	println("Pipeline", pipeline.ID, pipeline.Sha, pipeline.Ref, "SUCCESS")

	saveConfig(projectID, projectPath, config)

	_, resp, err = api.Commits.SetCommitStatus(projectID, pipeline.Sha, &gitlab.SetCommitStatusOptions{
		State:       gitlab.Success,
		Ref:         &pipeline.Ref,
		Name:        gitlab.String("pages:deploy"),
		Description: gitlab.String("deployed"),
	})
	panicOnAPIError(resp, err)
}

func configJob(message *workers.Msg, projectID int, projectPath string, config map[string]interface{}) {
	saveConfig(projectID, projectPath, config)
}
