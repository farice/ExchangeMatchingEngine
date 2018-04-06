package main

import (
	"bufio"
	"net"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Connections holds info about client and net connection
type Connection struct {
	conn   net.Conn
	Server *server
}

// TCP server
type server struct {
	address                  string // Address to open connection
	onNewConnectionCallback  func(c *Connection)
	onClientConnectionClosed func(c *Connection, err error)
	onNewMessage             func(c *Connection, message []byte)
}

// Read Connection data from channel
func (c *Connection) listen() {
	reader := bufio.NewReader(c.conn)

	// while(1)
	for {
		message_length, err := reader.ReadString('\n')
		if err != nil {
			c.conn.Close()
			c.Server.onClientConnectionClosed(c, err)
			return
		}

		message_length = strings.TrimSuffix(message_length, "\n")

		// message_length should indicate number of bytes of XML to read
		len_msg, err := strconv.Atoi(message_length)
		if err != nil {
			c.conn.Close()
			c.Server.onClientConnectionClosed(c, err)
			return
		}

		msg := make([]byte, len_msg)

		bytes_read, err := reader.Read(msg)
		// ensure that bytes read matches length specified of XML request
		if err != nil || bytes_read != len_msg {
			c.conn.Close()
			c.Server.onClientConnectionClosed(c, err)
			return
		}

		c.Server.onNewMessage(c, msg)
	}
}

// Send text message to Connection
func (c *Connection) Send(message string) error {
	defer LogMethodTimeElapsed("tcp_server.Send", time.Now())
	_, err := c.conn.Write([]byte(message))
	return err
}

// Send bytes to Connection
func (c *Connection) SendBytes(b []byte) error {
	defer LogMethodTimeElapsed("tcp_server.SendBytes", time.Now())
	_, err := c.conn.Write(b)
	return err
}

func (c *Connection) Conn() net.Conn {
	return c.conn
}

func (c *Connection) Close() error {
	return c.conn.Close()
}

// MARK: - Callbacks to handle new Connections, messages, closed connections

// Called right after server starts listening new Connection
func (s *server) OnNewConnection(callback func(c *Connection)) {
	s.onNewConnectionCallback = callback
}

// Called right after connection closed
func (s *server) OnClientConnectionClosed(callback func(c *Connection, err error)) {
	defer LogMethodTimeElapsed("tcp_server.OnClientConnectionClosed", time.Now())
	s.onClientConnectionClosed = callback
}

// Called when Connection receives new message
func (s *server) OnNewMessage(callback func(c *Connection, message []byte)) {
	defer LogMethodTimeElapsed("tcp_server.OnNewMessage", time.Now())
	s.onNewMessage = callback
}

// Start network server
func (s *server) Listen() {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		log.Fatal("Error starting TCP server.")
	}
	defer listener.Close()

	for {
		conn, _ := listener.Accept()
		client_connection := &Connection{
			conn:   conn,
			Server: s,
		}

		// lightweight thread managed by the Go runtime
		go client_connection.listen()
		s.onNewConnectionCallback(client_connection)
	}
}

// Creates new tcp server instance
func NewTCPServer(address string) *server {
	log.Info("Creating server with address: ", address)
	server := &server{
		address: address,
	}

	server.OnNewConnection(func(c *Connection) {})
	server.OnNewMessage(func(c *Connection, message []byte) {})
	server.OnClientConnectionClosed(func(c *Connection, err error) {})

	return server
}
