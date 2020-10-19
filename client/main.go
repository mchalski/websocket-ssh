package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	port, err := strconv.ParseInt(os.Getenv("SSH_PORT"), 10, 32)
	if err != nil {
		log.Fatal("provide SSH_PORT")
	}

	backend := os.Getenv("BACKEND_HOST")
	if backend == "" {
		log.Fatal("provide BACKEND_HOST")
	}

	signals := make(chan os.Signal, 1)
	stop := make(chan bool)
	signal.Notify(signals, os.Interrupt)
	go func() {
		for _ = range signals {
			fmt.Println("\nReceived an interrupt, stopping...")
			stop <- true
		}
	}()

	ssh := setupSSHServer(port)

	go func() {
		log.Println("starting ssh server")
		if err := ssh.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	var proxyConn net.Conn
	var perr error
	// allow some switching on Sleep until the server runs
	for retries := 5; retries > 0; retries-- {
		proxyConn, perr = setupProxy(port)

		if perr != nil {
			log.Println(err)
		} else {
			break
		}

		time.Sleep(1 * time.Second)
	}

	if perr != nil {
		log.Fatal(perr)
	}

	wsConn, err := setupWebsocket(backend)
	if err != nil {
		log.Fatal(err)
	}

	tunnel(proxyConn, wsConn)

	<-stop
}

func setupWebsocket(backend string) (*websocket.Conn, error) {
	log.Printf("starting websocket to %s\n", backend)
	conn, _, err := websocket.DefaultDialer.Dial("ws://"+backend+"/device/connect", nil)
	if err != nil {
		log.Fatal("dial:", err)
	}

	return conn, err
}

func tunnel(proxy net.Conn, ws *websocket.Conn) {
	log.Println("starting tcp proxy<->ws tunnel")

	// read input from WS
	go func() {
		for {
			_, message, err := ws.ReadMessage()
			log.Printf("DEBUG recv from websocket: %s", message)
			if err != nil {
				log.Fatal(err)
			}

			_, err = proxy.Write(message)
			log.Printf("DEBUG write to proxy: %s", message)
			if err != nil {
				log.Fatal("write to TCP err: %s", err.Error())
			}
		}
	}()

	// read input from tcp
	go func() {
		for {
			buf := make([]byte, 256)
			n, err := proxy.Read(buf)

			if err != nil {
				log.Fatal(err)
			}

			log.Printf("DEBUG recv from TCP: %s", buf[:n])

			err = ws.WriteMessage(websocket.TextMessage, buf[:n])
			if err != nil {
				log.Fatal(err)
				break
			}
			log.Printf("DEBUG write to WS: %s", buf[:n])
		}
	}()
}
