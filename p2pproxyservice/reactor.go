package p2pproxyservice

import (
	"time"

	"math/rand"

	clist "github.com/tendermint/tendermint/libs/clist"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/p2p"
	protop2p "github.com/tendermint/tendermint/proto/tendermint/p2p"
)

//add a const section with the channel ID dedicated to this reactor

const (
	ProxyP2PChannel = byte(0x50)
	maxMsgSize      = 1048576 // 1MB;

	// If a message fails wait this much before sending it again
	peerRetryMessageIntervalMS = 100
)

type GossipMessage interface {
	decreaseFanoutCount()
}

type NetworkGossipMessage struct {
	message      protop2p.ThresholdData
	fanoutCount  int
	receivedFrom p2p.Peer
}

var _ GossipMessage = &NetworkGossipMessage{}

func NewNetworkGossipMessage(message protop2p.ThresholdData, fanout int, peer p2p.Peer) *NetworkGossipMessage {
	m := &NetworkGossipMessage{
		message:      message,
		fanoutCount:  fanout,
		receivedFrom: peer,
	}

	return m
}

func (m *NetworkGossipMessage) decreaseFanoutCount() {
	m.fanoutCount = m.fanoutCount - 1
}

type ProxyP2PReactor struct {
	p2p.BaseReactor

	proxyP2P    *P2P_Proxy
	messagePool *clist.CList

	//for now let's not consider the logic to check the origin of the message

	//CList of messages to gossip
	//every CElement shouold be a new type gossipMessage
	//gossipMessage should have:
	//attributes
	//message (bytearray or proto.msg)
	//fanoutCount (integer)
	//methods
	//decreaseFanout

	//hashmap to register the messages that we already have
}

func NewProxyP2PReactor(proxy *P2P_Proxy) *ProxyP2PReactor {
	list := clist.New()
	reactor := &ProxyP2PReactor{
		//any additional fields
		//instanciate the CLIST
		proxyP2P:    proxy,
		messagePool: list,
	}
	reactor.BaseReactor = *p2p.NewBaseReactor("ProxyP2PReactor", reactor)
	return reactor
}

// SetLogger implements service.Service by setting the logger on the reactor.
func (reactor *ProxyP2PReactor) SetLogger(l log.Logger) {
	reactor.BaseService.Logger = l
}

func (reactor *ProxyP2PReactor) GetChannels() []*p2p.ChannelDescriptor {
	return []*p2p.ChannelDescriptor{
		{
			ID:                  ProxyP2PChannel,
			Priority:            6,
			SendQueueCapacity:   100,
			RecvBufferCapacity:  50 * 4096,
			RecvMessageCapacity: maxMsgSize,
			MessageType:         &protop2p.ThresholdData{}, //see if there is something so simple bytes
		},
	}
}

func (reactor *ProxyP2PReactor) ReceiveEnvelope(e p2p.Envelope) {
	println("[reactor-proxy] receive envelope ...")
	switch msg := e.Message.(type) {
	case *protop2p.ThresholdData:
		//check if the message is already in Clist
		//check if this node is the original sender (how? for sure we need a new field in the proto msg)
		//prevent loops for the gossiping
		//Send it to the proxy to send it to thetacrypt
		reactor.proxyP2P.ReceivedFromP2P(msg.GetData())
		//add the message to CList for gossip if the roundcount is not 0 yet (another field in the proto msg)
		gossipMsg := NewNetworkGossipMessage(*msg, 2, e.Src)
		reactor.messagePool.PushBack(*gossipMsg)
	default:
		println("[reactor-proxy] wrong message type")
	}
}
func (reactor *ProxyP2PReactor) AddPeer(peer p2p.Peer) {
	// here the light thread for handling the gossip should start
	go reactor.GossipMessageRoutine(peer)

}

// go routine for the peer --- GossipMessage
func (reactor *ProxyP2PReactor) GossipMessageRoutine(peer p2p.Peer) {
	var next *clist.CElement

	for {
		// This happens because the CElement we were looking at got garbage
		// collected (removed). That is, .NextWait() returned nil. Go ahead and
		// start from the beginning.
		if next == nil {
			select {
			case <-reactor.messagePool.WaitChan(): // Wait until evidence is available
				if next = reactor.messagePool.Front(); next == nil {
					continue
				}
			case <-peer.Quit():
				return
			case <-reactor.Quit():
				return
			}
		} else if !peer.IsRunning() || !reactor.IsRunning() {
			return
		}

		msg := next.Value.(NetworkGossipMessage)

		//decide at random if you have to send this message
		//the random variable should be fanout/number of nodes
		//IDEA: rand() < 0.7 ? 1 : 0
		//probToSend := float64(msg.fanoutCount) //add the division on the number of nodes
		random := rand.Float64()
		if random < 0.5 { //the probability to gossip or not
			if msg.message.HopCount > 1 && msg.message.SourceID != string(peer.NodeInfo().ID()) && msg.receivedFrom.ID() != peer.NodeInfo().ID() {

				thmsg := &protop2p.ThresholdData{
					Data:     msg.message.Data,
					HopCount: msg.message.HopCount - 1,
					SourceID: msg.message.SourceID,
				}

				env := p2p.Envelope{
					ChannelID: ProxyP2PChannel,
					Message:   thmsg,
				}
				success := p2p.SendEnvelopeShim(peer, env, reactor.BaseService.Logger) //why can't I use SendEnvelope here?

				if !success {
					println("[reactor-gossiping-message] not sending messages...")
					time.Sleep(peerRetryMessageIntervalMS * time.Millisecond)
					continue
				}

			}
		}

		//	afterCh := time.After(time.Second * broadcastEvidenceIntervalS) DO WE NEED THIS?
		select {
		// case <-afterCh:
		// 	// start from the beginning every tick.
		// 	// TODO: only do this if we're at the end of the list!
		// 	next = nil
		case <-next.NextWaitChan():
			// see the start of the for loop for nil check
			//check if next == tail (if true start again from the head)
			next = next.Next()
		case <-peer.Quit():
			return
		case <-reactor.Quit():
			return
		}
	}

	//For loop

	//here I can introduce the fanout

	//pick from the Clist and send to a peer (evry peer routine updates it's pointer next)
}
