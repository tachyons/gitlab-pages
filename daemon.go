package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
)

const (
	daemonRunProgram = "gitlab-pages-unprivileged"
)

func daemonMain() {
	if os.Args[0] != daemonRunProgram {
		return
	}

	// Validate that a working directory is valid
	// https://man7.org/linux/man-pages/man2/getcwd.2.html
	wd, err := os.Getwd()
	if err != nil {
		fatal(err, "could not get current working directory")
	} else if strings.HasPrefix(wd, "(unreachable)") {
		fatal(os.ErrPermission, "could not get current working directory")
	}

	logrus.WithFields(logrus.Fields{
		"uid": syscall.Getuid(),
		"gid": syscall.Getgid(),
		"wd":  wd,
	}).Info("starting the daemon as unprivileged user")

	// read the configuration from the pipe "ExtraFiles"
	var config config.Config
	if err := json.NewDecoder(os.NewFile(3, "options")).Decode(&config); err != nil {
		fatal(err, "could not decode app config")
	}
	runApp(&config)
	os.Exit(0)
}

func daemonReexec(uid, gid uint, args ...string) (cmd *exec.Cmd, err error) {
	path, err := os.Executable()
	if err != nil {
		return
	}

	cmd = &exec.Cmd{
		Path:   path,
		Args:   args,
		Env:    os.Environ(),
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		SysProcAttr: &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Uid: uint32(uid),
				Gid: uint32(gid),
			},
			Setsid: true,
		},
	}
	return
}

func daemonUpdateFd(cmd *exec.Cmd, fd uintptr) (childFd uintptr) {
	file := os.NewFile(fd, "[socket]")

	// we add 3 since, we have a 3 predefined FDs
	childFd = uintptr(3 + len(cmd.ExtraFiles))
	cmd.ExtraFiles = append(cmd.ExtraFiles, file)

	return
}

func daemonUpdateFds(cmd *exec.Cmd, fds []uintptr) {
	for idx, fd := range fds {
		fds[idx] = daemonUpdateFd(cmd, fd)
	}
}

func killProcess(cmd *exec.Cmd) {
	if cmd.Process != nil {
		cmd.Process.Kill()
	}
	cmd.Wait()
	for _, file := range cmd.ExtraFiles {
		file.Close()
	}
}

func passSignals(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGTERM, os.Interrupt, os.Kill)

	go func() {
		for cmd.Process != nil {
			cmd.Process.Signal(<-s)
		}
	}()
}

func daemonize(config *config.Config) error {
	uid := config.Daemon.UID
	gid := config.Daemon.GID
	pagesRoot := config.General.RootDir

	// Ensure pagesRoot is an absolute path. This will produce a different path
	// if any component of pagesRoot is a symlink (not likely). For example,
	// -pages-root=/some-path where ln -s /other-path /some-path
	// pagesPath will become: /other-path and we will fail to serve files from /some-path.
	// GitLab Rails also resolves the absolute path for `pages_path`
	// https://gitlab.com/gitlab-org/gitlab/blob/981ad651d8bd3690e28583eec2363a79f775af89/config/initializers/1_settings.rb#L296
	pagesRoot, err := filepath.Abs(pagesRoot)
	if err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"uid":        uid,
		"gid":        gid,
		"pages-root": pagesRoot,
	}).Info("running the daemon as unprivileged user")

	cmd, err := daemonReexec(uid, gid, daemonRunProgram)
	if err != nil {
		return err
	}
	defer killProcess(cmd)

	// Create a pipe to pass the configuration
	configReader, configWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	defer configWriter.Close()
	cmd.ExtraFiles = append(cmd.ExtraFiles, configReader)

	updateFds(config, cmd)

	// Start the process
	if err := cmd.Start(); err != nil {
		logrus.WithError(err).Error("start failed")
		return err
	}

	// Write the configuration
	if err := json.NewEncoder(configWriter).Encode(config); err != nil {
		return err
	}
	configWriter.Close()

	// Pass through signals
	passSignals(cmd)

	// Wait for process to exit
	return cmd.Wait()
}

func updateFds(config *config.Config, cmd *exec.Cmd) {
	for _, fds := range [][]uintptr{
		config.Listeners.HTTP,
		config.Listeners.HTTPS,
		config.Listeners.Proxy,
		config.Listeners.HTTPSProxyv2,
	} {
		daemonUpdateFds(cmd, fds)
	}

	for _, fdPtr := range []*uintptr{
		&config.ListenMetrics,
	} {
		if *fdPtr != 0 {
			*fdPtr = daemonUpdateFd(cmd, *fdPtr)
		}
	}
}
