package main

import (
	"fmt"
	"log"
	"net"
)

func setupProxy(sshPort int64) (net.Conn, error) {
	log.Println("connecting to proxy target")

	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", sshPort))

	if err != nil {
		return nil, err
	}
	log.Println("proxy connection established")

	return conn, nil
}
