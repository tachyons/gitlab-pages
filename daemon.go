package main

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"strconv"
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

func daemonReexec(cmdUser string, args ...string) (cmd *exec.Cmd, err error) {
	path, err := osext.Executable()
	if err != nil {
		return
	}

	u, err := user.Lookup(cmdUser)
	if err != nil {
		return
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return
	}

	gid, err := strconv.Atoi(u.Gid)
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

func daemonUpdateFd(cmd *exec.Cmd, fd *uintptr) {
	if *fd == 0 {
		return
	}

	file := os.NewFile(*fd, "[socket]")
	// we add 3 since, we have a 3 predefined FDs
	*fd = uintptr(3 + len(cmd.ExtraFiles))
	cmd.ExtraFiles = append(cmd.ExtraFiles, file)
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

func daemonize(config appConfig, cmdUser string) {
	var err error
	defer func() {
		if err != nil {
			log.Fatalln(err)
		}
	}()
	log.Printf("Running the daemon as unprivileged user: %v...", cmdUser)

	cmd, err := daemonReexec(cmdUser, daemonRunProgram)
	if err != nil {
		return
	}
	defer killProcess(cmd)

	// Create a pipe to pass the configuration
	configReader, configWriter, err := os.Pipe()
	if err != nil {
		return
	}
	defer configWriter.Close()
	cmd.ExtraFiles = append(cmd.ExtraFiles, configReader)

	// Create a new file and store the FD
	daemonUpdateFd(cmd, &config.ListenHTTP)
	daemonUpdateFd(cmd, &config.ListenHTTPS)
	daemonUpdateFd(cmd, &config.listenProxy)

	// Start the process
	if err = cmd.Start(); err != nil {
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
