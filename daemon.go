package main

import (
	"log"
	"os"
	"os/exec"
	"os/user"

	"fmt"
	"github.com/kardianos/osext"
	"strconv"
	"syscall"
)

func daemonize() {
	if *pagesUser == "" {
		return
	}

	path, err := osext.Executable()
	if err != nil {
		log.Fatalln(err)
	}

	u, err := user.Lookup(*pagesUser)
	if err != nil {
		log.Fatalln(err)
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		log.Fatalln(err)
	}

	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		log.Fatalln(err)
	}

	cmd := &exec.Cmd{
		Path:   path,
		Args:   append(os.Args, "-pages-user", "", "-pages-root", "/"),
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		SysProcAttr: &syscall.SysProcAttr{
			Chroot: *pagesRoot,
			Credential: &syscall.Credential{
				Uid: uint32(uid),
				Gid: uint32(gid),
			},
			//Setsid:     true,
			Setpgid: true,
		},
	}
	//cmd.SysProcAttr = nil

	fmt.Println("Deamonizing as", uid, "and", gid, "...")
	err = cmd.Run()
	if err != nil {
		log.Fatalln(err)
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
