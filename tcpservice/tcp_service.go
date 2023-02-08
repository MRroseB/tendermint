package tcpservice

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

const (
	HOST = "localhost"
	PORT = "8080"
	TYPE = "tcp"
)

// 4 bytes
const prefixSize = 4

type P2P_Proxy struct {
	//base.Service
	//config
	listener net.Listener
}

// pass in a config file the info about net addresses
func NewP2PProxy() *P2P_Proxy {
	listen, err := net.Listen(TYPE, HOST+":"+PORT)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	fmt.Println("Server is up and listening...")
	proxy := &P2P_Proxy{
		listener: listen,
	}
	return proxy
}

// OnStart -- Start handling request
func (proxy *P2P_Proxy) HandleIncomingRequests() {
	for {
		conn, err := proxy.listener.Accept()
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		fmt.Println("Server has accepted a request...")
		go handleRequest(conn) // go -- goroutine starts a lightweight thread that does the job
	}
}

func (proxy *P2P_Proxy) Send(msg []byte) {

}

func (proxy *P2P_Proxy) ReceiveFromP2P(msg []byte) {
	//Open the response channel
	tcpServer, err := net.ResolveTCPAddr(TYPE, HOST+":"+"8081")

	if err != nil {
		println("ResolveTCPAddr failed:", err.Error())
		os.Exit(1)
	}

	conn, err := net.DialTCP(TYPE, nil, tcpServer)
	if err != nil {
		println("Dial failed:", err.Error())
		os.Exit(1)
	}

	//create buffer for the msg
	buffer := createTcpBuffer([]byte(msg))
	_, err = conn.Write(buffer)
	if err != nil {
		println("Write data failed:", err.Error())
		os.Exit(1)
	}

	received := make([]byte, 1024)
	_, err = conn.Read(received)
	if err != nil {
		println("Read data failed:", err.Error())
		os.Exit(1)
	}

	println("Received message:", string(received))

	conn.Close() // Good practice to postpone the closure with defer
}

// func main() {
// 	proxy := NewP2PProxy()

// 	go proxy.HandleIncomingRequests()

// 	proxy.ReceiveFromP2P([]byte("Ciao")) //gestire gli errori

// 	for {
// 	}

// }

func handleRequest(conn net.Conn) {

	prefix := make([]byte, prefixSize) // incoming request size
	_, err := io.ReadFull(conn, prefix)
	if err != nil {
		log.Fatal(err)
	}
	totalDataLength := binary.BigEndian.Uint32(prefix[:])

	// Buffer to store the actual data
	buffer := make([]byte, totalDataLength-prefixSize)

	// Read actual data without prefix
	_, err = io.ReadFull(conn, buffer)
	if err != nil {
		log.Fatal(err)
	}

	//Broadcast on the p2p network

	// write data to response
	responseStr := fmt.Sprintf("Your message has been sent on the p2p")
	conn.Write([]byte(responseStr))

	// close conn
	defer conn.Close() // Good practice to postpone the closure with defer
}

// createTcpBuffer() implements the TCP protocol used in this application
// A stream of TCP data to be sent over has two parts: a prefix and the actual data itself
// The prefix is a fixed length byte that states how much data is being transferred over
func createTcpBuffer(data []byte) []byte {
	// Create a buffer with size enough to hold a prefix and actual data
	buf := make([]byte, prefixSize+len(data))

	// State the total number of bytes (including prefix) to be transferred over
	binary.BigEndian.PutUint32(buf[:prefixSize], uint32(prefixSize+len(data)))

	// Copy data into the remaining buffer
	copy(buf[prefixSize:], data[:])

	return buf
}
