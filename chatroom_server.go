package chatroom

import (
	"log"
	"net/http"

	"golang.org/x/net/websocket"
)

// The chatroom server structure.
type ChatServer struct {
	listenAddr     string
	password       string
	serverConnPool *connPool
}

// A connPool is used to store all the WebSocket connections, and utilizes channels for registering and unregistering them.
type connPool struct {
	connections []*websocket.Conn
	register    chan *websocket.Conn
	unregister  chan *websocket.Conn
}

// ChatServer constructor.
// "listenAddr" represents the address and port for handling requests.
// "password" means the requirement for users to provide a password. For a public chat server, the password can be empty.
func NewChatServer(listenAddr, password string) *ChatServer {
	chatServer := new(ChatServer)
	chatServer.listenAddr = listenAddr
	chatServer.password = password
	chatServer.serverConnPool = &connPool{
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
	return chatServer
}

// Uses channel to detect the register and unregister on connPool,
// Call this function with goroutine to avoid infinite loop.
func (c *connPool) execute() {
	// Infinite loop to catch register and unregister event.
	for {
		select {
		// Add WebSocket connection to the pool when catch register event.
		case r := <-c.register:
			c.connections = append(c.connections, r)
			log.Println("WebSocket connected,", r.Request().RemoteAddr, "register.")
			log.Println("Current connection pool:", c.GetPoolAddr())
		// Remove WebSocket connection from the pool when catch unregister event.
		case r := <-c.unregister:
			c.connections = removeConn(c.connections, r)
			log.Println("WebSocket disconnected,", r.Request().RemoteAddr, "unregister.")
			log.Println("Current connection pool:", c.GetPoolAddr())
		}
	}
}

// Retrieves all IP addresses of the connections in connPool.
func (c *connPool) GetPoolAddr() []string {
	var slice []string
	for _, ws := range c.connections {
		slice = append(slice, ws.Request().RemoteAddr)
	}
	return slice
}

// Removes the WebSocket connection elem from the slice and returns the modified slice.
// If elem does not exist in the slice, returns the original unchanged slice.
func removeConn(slice []*websocket.Conn, elem *websocket.Conn) []*websocket.Conn {
	var newSliceLen int
	if len(slice) <= 0 {
		newSliceLen = 0
	} else {
		newSliceLen = len(slice) - 1
	}
	newSlice := make([]*websocket.Conn, newSliceLen)
	for i, origElem := range slice {
		if origElem == elem {
			newSlice = append(slice[:i], slice[i+1:]...)
			return newSlice
		}
	}
	return slice
}

// When establishing a WebSocket connection, the server verifies the password and registers the client.
// If the password is incorrect, the registration process will be canceled and returned an error message to client.
func (s *ChatServer) registerServer(ws *websocket.Conn) {
	// Close WebSocket connextion before return.
	defer ws.Close()
	// Get chatroom password parameter form url.
	params := ws.Request().URL.Query()
	password := params.Get("pwd")
	// Check the password is correct or not,
	// if the chat server is public, skip password checking.
	if s.password == "" || s.password == password {
		// Register the connection to the ConnPool and continue listening.
		s.serverConnPool.register <- ws
		s.readMessage(ws)
	} else {
		log.Println(ws.Request().RemoteAddr, "Client connection failed: Incorrect password.")
		// TODO: send error message to client
	}
}

// A blocking function that continues listening for WebSocket messages.
// If the connection is disconnected, it should be unregistered from the ConnPool.
func (s *ChatServer) readMessage(ws *websocket.Conn) {
	var message string
	for {
		err := websocket.Message.Receive(ws, &message)
		if err != nil {
			s.serverConnPool.unregister <- ws
			log.Println(err)
			return
		}
		log.Println(ws.Request().RemoteAddr, ":", message)
		s.Broadcast(message)
	}
}

// Broadcast the message on the chat server ConnPool.
func (s *ChatServer) Broadcast(message string) (err error) {
	for _, ws := range s.serverConnPool.connections {
		if err := websocket.Message.Send(ws, message); err != nil {
			// Remove the connection from ConnPool
			s.serverConnPool.unregister <- ws
			log.Println(ws.Request().RemoteAddr, "disconnected :", err)
			return err
		}
	}
	return nil
}

// A blocking function that run the chat server.
func (s *ChatServer) Run() {
	// Listing ConnPool.
	go s.serverConnPool.execute()
	// TODO: Maybe support "/register" to a custom setting.
	// WebSocket handling.
	http.Handle("/register", websocket.Handler(s.registerServer))
	err := http.ListenAndServe(s.listenAddr, nil)
	if err != nil {
		log.Panic("ListenAndServe: " + err.Error())
	}
}
