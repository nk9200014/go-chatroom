package chatroom

import (
	"fmt"
	"log"
	"net/url"
	"time"

	"golang.org/x/net/websocket"
)

// ChatClient stores the server configuration and maintains the WebSocket connection to the server.
type ChatClient struct {
	ClientID   string
	conn       *websocket.Conn
	chatServer *ServerConfig
}

// ServerConfig stores the necessary information for connecting to the server
type ServerConfig struct {
	origin   string
	protocol string
	url_     *url.URL
}

// ChatClient constructor, you should construct a serverConfig first.
func NewChatClient(clientID string, sc *ServerConfig) *ChatClient {
	chatClient := new(ChatClient)
	chatClient.ClientID = clientID
	chatClient.chatServer = sc
	return chatClient
}

// ServerConfig constructor, parse the url and return ServerConfig if no errors occur.
func NewServerConfig(origin, protocol, url_string string) (serverConfig *ServerConfig, err error) {
	serverConfig = new(ServerConfig)
	serverConfig.origin = origin
	serverConfig.protocol = protocol
	url_, err := url.Parse(url_string)
	if err != nil {
		return nil, err
	}
	serverConfig.url_ = url_
	return serverConfig, nil
}

// TODO:Make the ClientID useful
// Register with the chat server,input the password if the server is not public.
func (c *ChatClient) Register(password string) {
	c.chatServer.url_.RawQuery = "pwd=" + password
	ws, err := websocket.Dial(c.chatServer.url_.String(), c.chatServer.protocol, c.chatServer.origin)
	if err != nil {
		log.Fatal(err)
	}
	c.conn = ws
	// A goroutine function that keep WebSocket alive.
	go keepWebsocketAlive(ws)
}

// TODO: Send the message with json
// Send the message to chat server, ensure you have registered with the server.
func (c *ChatClient) Send(message string) (err error) {
	if c.conn == nil {
		log.Println("Websocket connection do not establish, please register first.")
		return fmt.Errorf("Websocket connection do not establish, please register first.")
	} else if err := websocket.Message.Send(c.conn, message); err != nil {
		log.Println("Can not send message to server:", err)
		return fmt.Errorf("Can not send message to server: %v", err)
	}
	return nil
}

// TODO: Parse the message with json
// Read the message from chat server, ensure you have registered with the server.
func (c *ChatClient) Read() (message string, err error) {
	if c.conn == nil {
		log.Println("Websocket connection do not establish, please register first.")
		return "", fmt.Errorf("Websocket connection do not establish, please register first.")
	} else if err := websocket.Message.Receive(c.conn, &message); err != nil {
		log.Println("Can not receive message from server:", err)
		return "", fmt.Errorf("Can not receive message from server: %v", err)
	}
	return message, nil
}

// TODO: Maybe user can determine how oftn to sends a heartbeat message.
// A blocking function that continuously sends a heartbeat message to the server every 60 seconds.
func keepWebsocketAlive(ws *websocket.Conn) {
	defer ws.Close()
	for {
		time.Sleep(60 * time.Second)
		if err := websocket.Message.Send(ws, "heartbeat"); err != nil {
			log.Println("Can not send heartbeat to server:", err)
			return
		}
	}
}
