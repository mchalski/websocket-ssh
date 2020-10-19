package main

import (
	"bufio"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"net"

	"github.com/gorilla/websocket"
)

// the client accepts user's WS data via stdin
// internally translates them to an ssh stream and funnels them to an adapter listening on a tcp port
// (the adapter then handles repacking ssh data back into WS format, for consumption by the device proxy side)
// shuffles data to user's WS via stdout
func StartSSHClient(forwardTCPPort int, termWS *websocket.Conn, userKeyPath string) {
	key, err := ioutil.ReadFile(userKeyPath)
	if err != nil {
		log.Fatalf("unable to read private key: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("unable to parse private key: %v", err)
	}

	config := &ssh.ClientConfig{
		User: "backend-user",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},

		// not concerned with verifying the device end for now
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("localhost:%d", forwardTCPPort), config)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer client.Close()

	s, err := client.NewSession()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer s.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	log.Println("requesting PTY")
	if err := s.RequestPty("xterm", 40, 80, modes); err != nil {
		log.Fatal("request for pseudo terminal failed: ", err)
	}
	log.Println("have PTY")

	// Start remote shell
	log.Println("starting shell")
	p, _ := s.StdinPipe()
	pout, _ := s.StdoutPipe()

	input := make(chan []byte)
	output := make(chan []byte)

	scanner := bufio.NewScanner(pout)
	scanner.Split(bufio.ScanBytes)
	go func() {
		for {
			for scanner.Scan() {
				m := scanner.Bytes()
				log.Printf("recv from stdout: %s", string(m))
				output <- m
			}
		}
	}()

	// read input from WS
	go func() {
		for {
			mt, message, err := termWS.ReadMessage()
			log.Printf("recv from WS: %s, mt %d", message, mt)
			if err != nil {
				log.Fatal(err)
			}

			input <- message
		}
	}()

	go func() {
		for {
			select {
			case msgOut := <-output:
				err = termWS.WriteMessage(websocket.TextMessage, msgOut)
				log.Printf("write to WS: %s", msgOut)
				if err != nil {
					log.Fatal(err)
				}

			case msgIn := <-input:
				fmt.Fprint(p, string(msgIn))
				log.Printf("write to stdin: %s", msgIn)
			}
		}
	}()

	log.Println("started rw goroutines")
	if err := s.Run("/bin/bash"); err != nil {
		log.Fatal("failed to start shell: ", err)
	}
}
