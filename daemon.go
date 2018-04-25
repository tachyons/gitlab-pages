package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/kardianos/osext"
	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/jail"
)

const (
	daemonRunProgram = "gitlab-pages-unprivileged"

	pagesRootInChroot = "/pages"
)

func daemonMain() {
	if os.Args[0] != daemonRunProgram {
		return
	}

	log.WithFields(log.Fields{
		"uid": syscall.Getuid(),
		"gid": syscall.Getgid(),
	}).Info("starting the daemon as unprivileged user")

	// read the configuration from the pipe "ExtraFiles"
	var config appConfig
	if err := json.NewDecoder(os.NewFile(3, "options")).Decode(&config); err != nil {
		fatal(err)
	}
	runApp(config)
	os.Exit(0)
}

func daemonReexec(uid, gid uint, args ...string) (cmd *exec.Cmd, err error) {
	path, err := osext.Executable()
	if err != nil {
		return
	}

	cmd = &exec.Cmd{
		Path:   path,
		Args:   args,
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

	s := make(chan os.Signal)
	signal.Notify(s, syscall.SIGTERM, os.Interrupt, os.Kill)

	go func() {
		for cmd.Process != nil {
			cmd.Process.Signal(<-s)
		}
	}()
}

func daemonChroot(cmd *exec.Cmd) (*jail.Jail, error) {
	cage := jail.TimestampedJail("gitlab-pages", 0755)

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// Add /dev/urandom and /dev/random inside the jail. This is required to
	// support Linux versions < 3.17, which do not have the getrandom() syscall
	cage.MkDir("/dev", 0755)
	if err := cage.CharDev("/dev/urandom"); err != nil {
		return nil, err
	}

	if err := cage.CharDev("/dev/random"); err != nil {
		return nil, err
	}

	// Add gitlab-pages inside the jail
	err = cage.CopyTo("/gitlab-pages", cmd.Path)
	if err != nil {
		return nil, err
	}

	// Add /etc/resolv.conf inside the jail
	cage.MkDir("/etc", 0755)
	err = cage.Copy("/etc/resolv.conf")
	if err != nil {
		return nil, err
	}

	// Copy SSL_CERT_FILE inside the jail
	sslCertFile := os.Getenv("SSL_CERT_FILE")
	if sslCertFile != "" {
		cage.MkDir("/etc/ssl", 0755)
		err = cage.CopyTo("/etc/ssl/ca-bundle.pem", sslCertFile)
		if err != nil {
			return nil, err
		}
		cmd.Env = append(os.Environ(), "SSL_CERT_FILE=/etc/ssl/ca-bundle.pem")
	} else {
		log.Print("Missing SSL_CERT_FILE environment variable. HTTPS requests will fail")
	}

	// Bind mount shared folder
	cage.MkDir(pagesRootInChroot, 0755)
	cage.Bind(pagesRootInChroot, wd)

	// Update command to use chroot
	cmd.SysProcAttr.Chroot = cage.Path()
	cmd.Path = "/gitlab-pages"
	cmd.Dir = pagesRootInChroot

	err = cage.Build()
	if err != nil {
		return nil, err
	}

	return cage, nil
}

func daemonize(config appConfig, uid, gid uint) {
	var err error
	defer func() {
		if err != nil {
			fatal(err)
		}
	}()

	log.WithFields(log.Fields{
		"uid": uid,
		"gid": gid,
	}).Info("running the daemon as unprivileged user")

	cmd, err := daemonReexec(uid, gid, daemonRunProgram)
	if err != nil {
		return
	}
	defer killProcess(cmd)

	// Run daemon in chroot environment
	chroot, err := daemonChroot(cmd)
	if err != nil {
		log.WithError(err).Print("chroot failed")
		return
	}
	defer chroot.Dispose()

	// Create a pipe to pass the configuration
	configReader, configWriter, err := os.Pipe()
	if err != nil {
		return
	}
	defer configWriter.Close()
	cmd.ExtraFiles = append(cmd.ExtraFiles, configReader)

	// Create a new file and store the FD for each listener
	daemonUpdateFds(cmd, config.ListenHTTP)
	daemonUpdateFds(cmd, config.ListenHTTPS)
	daemonUpdateFds(cmd, config.ListenProxy)
	if config.ListenMetrics != 0 {
		config.ListenMetrics = daemonUpdateFd(cmd, config.ListenMetrics)
	}

	// Start the process
	if err = cmd.Start(); err != nil {
		log.WithError(err).Error("start failed")
		return
	}

	//detach binded mountpoints
	if err = chroot.LazyUnbind(); err != nil {
		log.WithError(err).Print("chroot lazy umount failed")
		return
	}

	// Write the configuration
	if err = json.NewEncoder(configWriter).Encode(config); err != nil {
		return
	}
	configWriter.Close()

	// Pass through signals
	passSignals(cmd)

	// Wait for process to exit
	err = cmd.Wait()
}
