package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
		fatal(err, "could not decode app config")
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

	s := make(chan os.Signal)
	signal.Notify(s, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	go func() {
		for cmd.Process != nil {
			cmd.Process.Signal(<-s)
		}
	}()
}

func chrootDaemon(cmd *exec.Cmd) (*jail.Jail, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	chroot := jail.Into(wd)

	// Generate a probabilistically-unique suffix for the copy of the pages
	// binary being placed into the chroot
	suffix := make([]byte, 16)
	if _, err := rand.Read(suffix); err != nil {
		return nil, err
	}

	tempExecutablePath := fmt.Sprintf("/.daemon.%x", suffix)

	if err := chroot.CopyTo(tempExecutablePath, cmd.Path); err != nil {
		return nil, err
	}

	// Update command to use chroot
	cmd.SysProcAttr.Chroot = chroot.Path()
	cmd.Path = tempExecutablePath
	cmd.Dir = "/"

	if err := chroot.Build(); err != nil {
		return nil, err
	}

	return chroot, nil
}

func jailCopyCertDir(cage *jail.Jail, sslCertDir, jailCertsDir string) error {
	log.WithFields(log.Fields{
		"ssl-cert-dir": sslCertDir,
	}).Debug("Copying certs from SSL_CERT_DIR")

	entries, err := ioutil.ReadDir(sslCertDir)
	if err != nil {
		return fmt.Errorf("failed to read SSL_CERT_DIR: %+v", err)
	}

	for _, fi := range entries {
		// Copy only regular files and symlinks
		mode := fi.Mode()
		if !(mode.IsRegular() || mode&os.ModeSymlink != 0) {
			continue
		}

		err = cage.CopyTo(jailCertsDir+"/"+fi.Name(), sslCertDir+"/"+fi.Name())
		if err != nil {
			log.WithError(err).Errorf("failed to copy cert: %q", fi.Name())
			// Go on and try to copy other files. We don't want the whole
			// startup process to fail due to a single failure here.
		}
	}

	return nil
}

func jailDaemonCerts(cmd *exec.Cmd, cage *jail.Jail) error {
	sslCertFile := os.Getenv("SSL_CERT_FILE")
	sslCertDir := os.Getenv("SSL_CERT_DIR")
	if sslCertFile == "" && sslCertDir == "" {
		log.Warn("Neither SSL_CERT_FILE nor SSL_CERT_DIR environment variable is set. HTTPS requests will fail.")
		return nil
	}

	// This assumes cage.MkDir("/etc") has already been called
	cage.MkDir("/etc/ssl", 0755)

	// Copy SSL_CERT_FILE inside the jail
	if sslCertFile != "" {
		jailCertsFile := "/etc/ssl/ca-bundle.pem"
		err := cage.CopyTo(jailCertsFile, sslCertFile)
		if err != nil {
			return fmt.Errorf("failed to copy SSL_CERT_FILE: %+v", err)
		}
		cmd.Env = append(cmd.Env, "SSL_CERT_FILE="+jailCertsFile)
	}

	// Copy all files and symlinks from SSL_CERT_DIR into the jail
	if sslCertDir != "" {
		jailCertsDir := "/etc/ssl/certs"
		cage.MkDir(jailCertsDir, 0755)
		err := jailCopyCertDir(cage, sslCertDir, jailCertsDir)
		if err != nil {
			return err
		}
		cmd.Env = append(cmd.Env, "SSL_CERT_DIR="+jailCertsDir)
	}

	return nil
}

func jailDaemon(cmd *exec.Cmd) (*jail.Jail, error) {
	cage := jail.CreateTimestamped("gitlab-pages", 0755)

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

	// Add certificates inside the jail
	err = jailDaemonCerts(cmd, cage)
	if err != nil {
		return nil, err
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

func daemonize(config appConfig, uid, gid uint, inPlace bool) error {
	log.WithFields(log.Fields{
		"uid":      uid,
		"gid":      gid,
		"in-place": inPlace,
	}).Info("running the daemon as unprivileged user")

	cmd, err := daemonReexec(uid, gid, daemonRunProgram)
	if err != nil {
		return err
	}
	defer killProcess(cmd)

	// Run daemon in chroot environment
	var wrapper *jail.Jail
	if inPlace {
		wrapper, err = chrootDaemon(cmd)
	} else {
		wrapper, err = jailDaemon(cmd)
	}
	if err != nil {
		log.WithError(err).Print("chroot failed")
		return err
	}
	defer wrapper.Dispose()

	// Create a pipe to pass the configuration
	configReader, configWriter, err := os.Pipe()
	if err != nil {
		return err
	}
	defer configWriter.Close()
	cmd.ExtraFiles = append(cmd.ExtraFiles, configReader)

	updateFds(&config, cmd)

	// Start the process
	if err := cmd.Start(); err != nil {
		log.WithError(err).Error("start failed")
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

func updateFds(config *appConfig, cmd *exec.Cmd) {
	for _, fds := range [][]uintptr{
		config.ListenHTTP,
		config.ListenHTTPS,
		config.ListenProxy,
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
