package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/kardianos/osext"
)

const daemonRunProgram = "gitlab-pages-unprivileged"

func daemonMain() {
	if os.Args[0] != daemonRunProgram {
		return
	}

	log.Printf("Starting the daemon as unprivileged user (uid: %d, gid: %d)...", syscall.Getuid(), syscall.Getgid())

	// read the configuration from the pipe "ExtraFiles"
	var config appConfig
	if err := json.NewDecoder(os.NewFile(3, "options")).Decode(&config); err != nil {
		log.Fatalln(err)
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

func copyFile(dest, src string, perm os.FileMode) (err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return
}

func daemonCreateExecutable(name string, perm os.FileMode) (file *os.File, err error) {
	// We assume that crypto random generates true random, non-colliding hash
	b := make([]byte, 16)
	_, err = rand.Read(b)
	if err != nil {
		return
	}

	path := fmt.Sprintf("%s.%x", name, b)
	file, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return
	}
	return
}

func daemonChroot(cmd *exec.Cmd) (path string, err error) {
	wd, err := os.Getwd()
	if err != nil {
		return
	}

	// Generate temporary file
	temporaryExecutable, err := daemonCreateExecutable(".daemon", 0755)
	if err != nil {
		return
	}
	defer temporaryExecutable.Close()
	defer func() {
		// Remove the temporary file in case of failure
		if err != nil {
			os.Remove(temporaryExecutable.Name())
		}
	}()

	// Open current executable
	executableFile, err := os.Open(cmd.Path)
	if err != nil {
		return
	}
	defer executableFile.Close()

	// Copy the executable content
	_, err = io.Copy(temporaryExecutable, executableFile)
	if err != nil {
		return
	}

	// Update command to use chroot
	cmd.SysProcAttr.Chroot = wd
	cmd.Path = temporaryExecutable.Name()
	cmd.Dir = "/"
	path = filepath.Join(wd, temporaryExecutable.Name())
	return
}

func daemonize(config appConfig, uid, gid uint) {
	var err error
	defer func() {
		if err != nil {
			log.Fatalln(err)
		}
	}()
	log.Printf("Running the daemon as unprivileged user (uid:%d, gid: %d)...", uid, gid)

	cmd, err := daemonReexec(uid, gid, daemonRunProgram)
	if err != nil {
		return
	}
	defer killProcess(cmd)

	// Run daemon in chroot environment
	temporaryExecutable, err := daemonChroot(cmd)
	if err != nil {
		println("Chroot failed", err)
		return
	}
	defer os.Remove(temporaryExecutable)

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
		println("Start failed", err)
		return
	}

	// Write the configuration
	if err = json.NewEncoder(configWriter).Encode(config); err != nil {
		return
	}
	configWriter.Close()

	// Remove executable
	os.Remove(temporaryExecutable)

	// Pass through signals
	passSignals(cmd)

	// Wait for process to exit
	err = cmd.Wait()
}
