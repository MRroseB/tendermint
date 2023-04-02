package p2pproxyservice

import (
	"fmt"
	"net"
	"os"

	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/service"
	"github.com/tendermint/tendermint/p2p"
	protop2p "github.com/tendermint/tendermint/proto/tendermint/p2p"
)

const (
	HOST = "0.0.0.0"
	PORT = "8080"
	TYPE = "tcp"
)

// 4 bytes
const prefixSize = 4

type P2P_Proxy struct {
	service.BaseService
	//config
	haddr         string
	listeningAddr string
	sw            *p2p.Switch
}

// pass in a config file the info about net addresses
func NewP2PProxy(haddr string, listeningAddr string, logger log.Logger, sw *p2p.Switch) *P2P_Proxy {

	proxy := &P2P_Proxy{
		haddr:         haddr,
		listeningAddr: listeningAddr,
		sw:            sw,
	}

	proxy.BaseService = *service.NewBaseService(logger, "P2PProxy", proxy)
	return proxy
}

// OnStart -- start the Listener in a subroutine
func (proxy *P2P_Proxy) OnStart() error {

	// go proxy.StartListener() // see how to handle catching the error
	// don't use go here

	return nil
}

func (proxy *P2P_Proxy) OnStop() {

	// stop the listner

	//handle the error
}

func (proxy P2P_Proxy) StartListener() error {

	listener, err := net.Listen(TYPE, proxy.haddr+":"+PORT)
	if err != nil {
		return err
	}
	fmt.Println("Server is up and listening...")

	go proxy.ReceiveFromThetacrypt(listener)
	return nil
}

func (proxy *P2P_Proxy) ReceiveFromThetacrypt(listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		fmt.Println("Server has accepted a request...")
		go proxy.handleRequest(conn) // go -- goroutine starts a lightweight thread that does the job
	}
}

func (proxy *P2P_Proxy) SendToPeers(msg []byte) {
	// peers := proxy.sw.Peers().List()
	// successChan := make(chan bool, len(peers))
	//send the msg on the p2p with sw.Broadcast

	sourceID := proxy.sw.NodeInfo().ID()
	thmsg := &protop2p.ThresholdData{
		Data:     msg,
		HopCount: 2,
		SourceID: string(sourceID),
	}
	successChan := proxy.sw.BroadcastEnvelope(
		p2p.Envelope{
			ChannelID: ProxyP2PChannel,
			Message:   thmsg,
		}) //use broadcast envelope

	for success := range successChan {
		if !success {
			println("[tcp-service] not sending messages...")
		}
	}
}

func (proxy *P2P_Proxy) SendToThetacrypt(msg []byte) error {

	println("[tcp_service] Received message from reactor:", string(msg))

	//Open a tcp connection and send the msg
	tcpServer, err := net.ResolveTCPAddr(TYPE, proxy.listeningAddr+":8081")

	if err != nil {
		println("ResolveTCPAddr failed:", err.Error())
		os.Exit(1)
		return err
	}

	conn, err := net.DialTCP(TYPE, nil, tcpServer)
	if err != nil {
		println("Dial failed:", err.Error())
		os.Exit(1)
		return err
	}

	_, err = conn.Write(msg)
	if err != nil {
		println("Write data failed:", err.Error())
		os.Exit(1)
		return err
	}

	return nil
}

func (proxy *P2P_Proxy) handleRequest(conn net.Conn) error {

	buffer := make([]byte, 4096) //Decide the dimension of the vector

	//Read from the stream
	_, err := conn.Read(buffer)
	if err != nil {
		println("Error reading buffer ...")
		return err
	}

	//Broadcast on the p2p network
	proxy.SendToPeers(buffer)

	// close conn
	defer conn.Close() // Good practice to postpone the closure with defer

	return nil
}
