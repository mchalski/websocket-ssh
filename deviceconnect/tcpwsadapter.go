package main

import (
	"fmt"
	"log"
	"net"

	"github.com/gorilla/websocket"
)

// TCPWSAdapter proxies between an ad hoc tcp connection
// and a websocket (in both directions)
// this is to bridge a component speaking tcp (crypto/ssh client)
// and the client proxy on the far end, speaking WS
func StartTCPWSAdapter(wsConn *websocket.Conn, port int) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))

	if err != nil {
		return err
	}

	conn, err := listener.Accept()
	if err != nil {
		return err
	}
	log.Printf("Accepted connection from %v\n", conn.RemoteAddr().String())

	forward(conn, wsConn)
	return nil
}

func forward(conn net.Conn, wsConn *websocket.Conn) error {
	recvBuf := make([]byte, 1024)

	// accept tcp, frame to websocket
	go func() error {
		for {
			n, err := conn.Read(recvBuf)
			if err != nil {
				return err
			}
			log.Printf("recv from tcp: %s", recvBuf[:n])

			wsConn.WriteMessage(websocket.TextMessage, recvBuf[:n])
		}
	}()
	// accept websocket, tunnel to tcp
	go func() error {
		for {
			_, message, err := wsConn.ReadMessage()
			if err != nil {
				return err
			}

			log.Printf("recv from WS: %s", message)
			conn.Write(message)
		}
	}()

	return nil
}
