package main

import (
	//"golang.org/x/crypto/ssh"
	"log"
	"net/http"
	//"time"

	"github.com/gorilla/websocket"
)

// some global state - it's just 1 user and 1 device
var termConn *websocket.Conn
var devConn *websocket.Conn
var userKeyPath = "/home/backend-user/.ssh/id_rsa"

func main() {
	http.HandleFunc("/device/connect", DevConnectHandler)
	http.HandleFunc("/user/terminal", UserTerminalHandler)
	http.Handle("/", http.FileServer(http.Dir("/root/assets")))
	log.Fatal(http.ListenAndServe("0.0.0.0:80", nil))
}

// intializes device WS connection
func DevConnectHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("GET /device/connect")

	upgrader := websocket.Upgrader{}

	var err error
	devConn, err = upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	log.Println("upgraded device to websockets")
}

// handles user's terminal WS session
func UserTerminalHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("GET /user/terminal")

	upgrader := websocket.Upgrader{}
	var err error

	termConn, err = upgrader.Upgrade(w, r, nil)
	log.Println("upgraded user to wesockets")

	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	port := getFreePort()
	go StartTCPWSAdapter(devConn, port)
	go StartSSHClient(port, termConn, userKeyPath)
}

func getFreePort() int {
	// pretend we're selecting a random, free port
	// e.g. https://github.com/phayes/freeport
	return 33333
}
