package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"github.com/gliderlabs/ssh"
	"github.com/kr/pty"
)

var authorizedKeysPath = "/root/ssh/authorized_keys"

func setupSSHServer(port int64) ssh.Server {
	server := ssh.Server{
		Addr: fmt.Sprintf(":%d", port),
		Handler: ssh.Handler(func(s ssh.Session) {
			cmd := exec.Command("/bin/bash")
			ptyReq, winCh, isPty := s.Pty()
			if isPty {
				cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
				f, err := pty.Start(cmd)
				if err != nil {
					panic(err)
				}
				go func() {
					for win := range winCh {
						setWinsize(f, win.Width, win.Height)
					}
				}()
				go func() {
					io.Copy(f, s) // stdin
				}()
				io.Copy(s, f) // stdout
			} else {
				io.WriteString(s, "No PTY requested.\n")
				s.Exit(1)
			}
		}),
	}

	authorizedKeys, err := readAuthorizedKeys()
	if err != nil {
		log.Fatal(err)
	}

	authOpt := ssh.PublicKeyAuth(
		func(ctx ssh.Context, key ssh.PublicKey) bool {
			for _, authorizedKey := range authorizedKeys {
				if ssh.KeysEqual(key, authorizedKey) {
					return true
				}
			}
			return false
		})

	err = server.SetOption(authOpt)
	if err != nil {
		log.Fatal(err)
	}

	return server
}

func readAuthorizedKeys() ([]ssh.PublicKey, error) {
	// read authorized_keys file
	bytes, err := ioutil.ReadFile(authorizedKeysPath)
	if err != nil {
		log.Fatal(err)
	}

	var authorizedKeys []ssh.PublicKey
	rest := bytes

	for {
		var authorizedKey ssh.PublicKey
		var err error
		authorizedKey, _, _, rest, err = ssh.ParseAuthorizedKey(rest)
		if err != nil {
			return nil, err
		}

		authorizedKeys = append(authorizedKeys, authorizedKey)

		if len(strings.TrimSpace(string(rest))) == 0 {
			break
		}
	}

	return authorizedKeys, nil
}

func setWinsize(f *os.File, w, h int) {
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
}
