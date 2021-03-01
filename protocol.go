package main

import (
	"fmt"
	"net"
	"reflect"
	"sort"
	"time"

	"github.com/nm-morais/go-babel/pkg/errors"
	"github.com/nm-morais/go-babel/pkg/logs"
	"github.com/nm-morais/go-babel/pkg/message"
	"github.com/nm-morais/go-babel/pkg/nodeWatcher"
	"github.com/nm-morais/go-babel/pkg/notification"
	"github.com/nm-morais/go-babel/pkg/peer"
	"github.com/nm-morais/go-babel/pkg/protocol"
	"github.com/nm-morais/go-babel/pkg/protocolManager"
	"github.com/nm-morais/go-babel/pkg/timer"
	"github.com/sirupsen/logrus"
)

const (
	protoID = 1000
	name    = "CyclonTMan"
)

type CyclonTManConfig struct {
	SelfPeer struct {
		AnalyticsPort int    `yaml:"analyticsPort"`
		Port          int    `yaml:"port"`
		Host          string `yaml:"host"`
	} `yaml:"self"`
	BootstrapPeers []struct {
		Port          int    `yaml:"port"`
		Host          string `yaml:"host"`
		AnalyticsPort int    `yaml:"analyticsPort"`
	} `yaml:"bootstrapPeers"`

	DialTimeoutMiliseconds int    `yaml:"dialTimeoutMiliseconds"`
	LogFolder              string `yaml:"logFolder"`
	CacheViewSize          int    `yaml:"cacheViewSize"`
	TManViewSize           int    `yaml:"tManViewSize"`
	TManFanout             int    `yaml:"tManFanout"`
	ShuffleTimeSeconds     int    `yaml:"shuffleTimerSeconds"`
	TManTimerSeconds       int    `yaml:"tManTimerSeconds"`
	L                      int    `yaml:"l"`
}
type CyclonTMan struct {
	babel                  protocolManager.ProtocolManager
	nodeWatcher            nodeWatcher.NodeWatcher
	logger                 *logrus.Logger
	conf                   *CyclonTManConfig
	selfIsBootstrap        bool
	bootstrapNodes         []peer.Peer
	cyclonView             *View
	tManView               *View
	measuringNodes         map[string]bool
	pendingCyclonExchanges map[string][]*PeerState
}

func NewCyclonTManProtocol(babel protocolManager.ProtocolManager, nw nodeWatcher.NodeWatcher, conf *CyclonTManConfig) protocol.Protocol {
	logger := logs.NewLogger(name)
	selfIsBootstrap := false
	bootstrapNodes := []peer.Peer{}
	for _, p := range conf.BootstrapPeers {
		boostrapNode := peer.NewPeer(net.ParseIP(p.Host), uint16(p.Port), uint16(p.AnalyticsPort))
		bootstrapNodes = append(bootstrapNodes, boostrapNode)
		if peer.PeersEqual(babel.SelfPeer(), boostrapNode) {
			selfIsBootstrap = true
		}
	}

	logger.Infof("Starting with selfPeer:= %+v", babel.SelfPeer())
	logger.Infof("Starting with bootstraps:= %+v", bootstrapNodes)
	logger.Infof("Starting with selfIsBootstrap:= %+v", selfIsBootstrap)

	return &CyclonTMan{
		babel:                  babel,
		nodeWatcher:            nw,
		logger:                 logger,
		conf:                   conf,
		selfIsBootstrap:        selfIsBootstrap,
		bootstrapNodes:         bootstrapNodes,
		cyclonView:             &View{capacity: conf.CacheViewSize, asArr: []*PeerState{}, asMap: map[string]*PeerState{}},
		tManView:               &View{capacity: conf.TManViewSize, asArr: []*PeerState{}, asMap: map[string]*PeerState{}},
		measuringNodes:         make(map[string]bool),
		pendingCyclonExchanges: make(map[string][]*PeerState),
	}
}

func (c *CyclonTMan) ID() protocol.ID {
	return protoID
}

func (c *CyclonTMan) Name() string {
	return name
}

func (c *CyclonTMan) Logger() *logrus.Logger {
	return c.logger
}

func (c *CyclonTMan) Init() {
	// CYCLON
	c.babel.RegisterTimerHandler(protoID, ShuffleTimerID, c.HandleShuffleTimer)
	c.babel.RegisterMessageHandler(protoID, &ShuffleMessage{}, c.HandleShuffleMessage)
	c.babel.RegisterMessageHandler(protoID, &ShuffleMessageReply{}, c.HandleShuffleMessageReply)

	// T-MAN
	c.babel.RegisterTimerHandler(protoID, GossipTimerID, c.HandleGossipTimer)
	c.babel.RegisterMessageHandler(protoID, &TManGossipMsg{}, c.HandleTManGossipMessage)
	c.babel.RegisterMessageHandler(protoID, tManGossipMsgReply{}, c.HandleTManGossipMessageReply)
	c.babel.RegisterNotificationHandler(protoID, PeerMeasuredNotification{}, c.handlePeerMeasuredNotification)
}

func (c *CyclonTMan) Start() {
	c.logger.Infof("Starting with confs: %+v", c.conf)
	if !c.selfIsBootstrap {
		for _, bootstrap := range c.bootstrapNodes {
			c.cyclonView.add(&PeerState{
				Peer: bootstrap,
				age:  0,
			}, false)
		}
	}
	c.babel.RegisterPeriodicTimer(c.ID(), ShuffleTimer{duration: time.Duration(c.conf.ShuffleTimeSeconds) * time.Second})
	c.babel.RegisterPeriodicTimer(c.ID(), GossipTimer{time.Duration(c.conf.TManTimerSeconds) * time.Second})
}

// ---------------- Cyclon----------------

func (c *CyclonTMan) HandleShuffleTimer(t timer.Timer) {
	if c.cyclonView.size() == 0 {
		c.logger.Info("Returning due to having no neighbors")
		return
	}

	for _, p := range c.cyclonView.asArr {
		p.age++
	}
	viewAsArr := c.cyclonView.asArr
	sort.Sort(viewAsArr)
	q := viewAsArr[0]
	c.logger.Infof("Oldest level peer: %s:%d", q.Peer.String(), q.age)
	if _, ok := c.pendingCyclonExchanges[q.String()]; ok {
		return //TODO
	}
	subset := append(c.cyclonView.getRandomElementsFromView(c.conf.L-1, q), &PeerState{
		Peer: c.babel.SelfPeer(),
		age:  0,
	})
	c.pendingCyclonExchanges[q.String()] = subset
	c.cyclonView.remove(q)
	toSend := NewShuffleMsg(subset)
	c.sendMessageTmpTransport(toSend, q)
	c.logger.Infof("Sending shuffle message %+v to %s", toSend, q)
}

func (c *CyclonTMan) HandleShuffleMessage(sender peer.Peer, msg message.Message) {
	shuffleMsg := msg.(*ShuffleMessage)
	c.logger.Infof("Received shuffle message %+v from %s", shuffleMsg, sender)
	peersToReply := c.cyclonView.getRandomElementsFromView(len(shuffleMsg.peers), shuffleMsg.peers...)
	toSend := NewShuffleMsgReply(peersToReply)
	c.sendMessageTmpTransport(toSend, sender)
	c.mergeCyclonViewWith(shuffleMsg.ToPeerStateArr(), peersToReply, sender)
}

func (c *CyclonTMan) HandleShuffleMessageReply(sender peer.Peer, msg message.Message) {
	shuffleMsgReply := msg.(*ShuffleMessageReply)
	c.logger.Infof("Received shuffle reply message %+v from %s", shuffleMsgReply, sender)
	c.mergeCyclonViewWith(shuffleMsgReply.ToPeerStateArr(), c.pendingCyclonExchanges[sender.String()], sender)
	delete(c.pendingCyclonExchanges, sender.String())
}

func (c *CyclonTMan) mergeCyclonViewWith(sample []*PeerState, sentPeers []*PeerState, sender peer.Peer) {
	for _, p := range sample {
		if peer.PeersEqual(c.babel.SelfPeer(), p.Peer) {
			continue // discard all entries pointing to self
		}

		if c.cyclonView.contains(p) {
			continue
		}

		// if theere is space, just add
		if !c.cyclonView.isFull() {
			c.cyclonView.add(p, false)
			continue
		}

		// attempt to drop sent peers, if there is any
		for len(sentPeers) > 0 {
			first := sentPeers[0]
			if c.cyclonView.contains(first) {
				c.cyclonView.remove(first)
				c.cyclonView.add(p, false)
				continue
			}
			sentPeers = sentPeers[1:]
		}

		c.cyclonView.add(p, true)
		if c.tManView.size() == 0 && c.cyclonView.size() >= c.conf.TManViewSize && len(c.measuringNodes) == 0 { // initialize T-Man
			rndSample := c.cyclonView.getRandomElementsFromView(c.conf.TManViewSize)
			c.issueMeasurementsFor(rndSample.ToPeerArr())
			return
		}
	}
	c.logCyclonTManState()
}

// ---------------- t-MAN ----------------

func (c *CyclonTMan) HandleGossipTimer(t timer.Timer) {
	c.logger.Info("Gossip timer trigger")

	if c.tManView.size() == 0 {
		c.logger.Info("Returning brecuase tMan view is 0")
		return
	}

	sort.SliceStable(c.tManView.asArr, func(i, j int) bool { return c.tManView.asArr[i].age < c.tManView.asArr[j].age })
	p := c.tManView.asArr[0]
	toSend := NewTManGossipMsg(c.makeTManBuf())
	c.sendMessageTmpTransport(toSend, p)
	c.logger.Infof("Sending t-man gossip message %+v to %s", toSend, p)
}

// array must be sorted
func (c *CyclonTMan) makeTManBuf() []peer.Peer {
	nrPeersToSend := c.conf.TManFanout
	if len(c.tManView.asArr)-1 < nrPeersToSend {
		nrPeersToSend = len(c.tManView.asArr) - 1
	}
	buffer := []peer.Peer{}
	if len(c.tManView.asArr) > 0 && nrPeersToSend > 0 {
		buffer = append(buffer, c.tManView.asArr[1:nrPeersToSend].ToPeerArr()...)
	}
	buffer = append(buffer, c.babel.SelfPeer())
	rndElems := c.cyclonView.getRandomElementsFromView(c.conf.TManFanout, buffer...)
	for _, p := range rndElems {
		buffer = append(buffer, p.Peer)
	}
	return buffer
}

func (c *CyclonTMan) HandleTManGossipMessage(sender peer.Peer, msg message.Message) {
	gossipMsg := msg.(*TManGossipMsg)
	c.logger.Infof("Received TMan gossip message %+v from %s", gossipMsg, sender)
	c.issueMeasurementsFor(gossipMsg.peers)
	sort.Sort(c.tManView.asArr)
	nrPeersToSend := c.conf.TManFanout
	if len(c.tManView.asArr) < nrPeersToSend {
		nrPeersToSend = len(c.tManView.asArr)
	}
	toSend := NewTManGossipMsgReply(c.tManView.asArr[:nrPeersToSend])
	c.sendMessageTmpTransport(toSend, sender)
	c.logger.Infof("Sending t-man gossip message reply %+v to %s", toSend, sender)
}

func (c *CyclonTMan) HandleTManGossipMessageReply(sender peer.Peer, msg message.Message) {
	gossipMsgReply := msg.(tManGossipMsgReply)
	c.logger.Infof("Received TMan gossip message reply %+v from %s", gossipMsgReply, sender)
	c.issueMeasurementsFor(gossipMsgReply.peers)
}

func (c *CyclonTMan) handlePeerMeasuredNotification(n notification.Notification) {
	peerMeasuredNotification := n.(PeerMeasuredNotification)
	peerMeasured := peerMeasuredNotification.peerMeasured
	peerMeasuredNInfo, err := c.nodeWatcher.GetNodeInfo(peerMeasured)
	defer c.nodeWatcher.Unwatch(peerMeasured, c.ID())
	defer delete(c.measuringNodes, peerMeasured.String())
	if err != nil {
		c.logger.Errorf("peer was %s not being measured", peerMeasured.String())
		return
	}
	for _, p2 := range c.tManView.asArr {
		if peer.PeersEqual(peerMeasured, p2) {
			return
		}
	}
	c.logger.Infof("Peer measured: %s:%+v", peerMeasured.String(), peerMeasuredNInfo.LatencyCalc().CurrValue())
	// measurements such that active peers have known costs
	measuredScore := peerMeasuredNInfo.LatencyCalc().CurrValue().Milliseconds()
	aux := &PeerState{
		Peer: peerMeasured,
		age:  uint16(measuredScore),
	}
	c.tManView.asArr = append(c.tManView.asArr, aux)
	sort.SliceStable(c.tManView.asArr, func(i, j int) bool { return c.tManView.asArr[i].age < c.tManView.asArr[j].age })
	if len(c.tManView.asArr) > c.tManView.capacity {
		c.tManView.asArr = c.tManView.asArr[:c.tManView.capacity]
	}
	c.logTManState()
}

func (c *CyclonTMan) issueMeasurementsFor(peers []peer.Peer) {
outer:
	for _, p := range peers {
		if _, ok := c.measuringNodes[p.String()]; ok {
			continue
		}
		if peer.PeersEqual(p, c.babel.SelfPeer()) {
			continue
		}
		for _, p2 := range c.tManView.asArr {
			if peer.PeersEqual(p, p2) {
				continue outer
			}
		}
		c.nodeWatcher.Watch(p, c.ID())
		condition := nodeWatcher.Condition{
			Repeatable:                false,
			CondFunc:                  func(nodeWatcher.NodeInfo) bool { return true },
			EvalConditionTickDuration: 500 * time.Millisecond,
			Notification:              NewPeerMeasuredNotification(p),
			Peer:                      p,
			EnableGracePeriod:         false,
			ProtoId:                   c.ID(),
		}
		c.nodeWatcher.NotifyOnCondition(condition)
		c.measuringNodes[p.String()] = true
	}
}

// ---------------- timer handlers ----------------

func (c *CyclonTMan) HandleDebugTimer(t timer.Timer) {

}

// ---------------- Networking Handlers ----------------

func (c *CyclonTMan) InConnRequested(dialerProto protocol.ID, p peer.Peer) bool {
	if dialerProto != c.ID() {
		c.logger.Warnf("Denying connection from peer %+v", p)
		return false
	}
	return false
}

func (c *CyclonTMan) OutConnDown(p peer.Peer) {
	c.logger.Errorf("Peer %s out connection went down", p.String())
}

func (c *CyclonTMan) DialFailed(p peer.Peer) {
	c.logger.Errorf("Failed to dial peer %s", p.String())
}

func (c *CyclonTMan) DialSuccess(sourceProto protocol.ID, p peer.Peer) bool {
	return false
}

func (c *CyclonTMan) MessageDelivered(msg message.Message, p peer.Peer) {
	c.logger.Infof("Message of type [%s] was delivered to %s", reflect.TypeOf(msg), p.String())
}

func (c *CyclonTMan) MessageDeliveryErr(msg message.Message, p peer.Peer, err errors.Error) {
	c.logger.Warnf("Message %s was not sent to %s because: %s", reflect.TypeOf(msg), p.String(), err.Reason())
}

// ---------------- Auxiliary functions ----------------

func (c *CyclonTMan) logCyclonTManState() {
	c.logger.Info("------------- Cyclon state -------------")
	var toLog string
	toLog = "Cyclon view : "

	for idx, p := range c.cyclonView.asArr {
		toLog += fmt.Sprintf("%s:%d", p.String(), p.age)
		if idx < len(c.cyclonView.asArr)-1 {
			toLog += ", "
		}
	}
	c.logger.Info(toLog)
}

func (c *CyclonTMan) logTManState() {
	c.logger.Info("------------- TMan state -------------")
	var toLog string
	toLog = "T-Man view : "

	for idx, p := range c.tManView.asArr {
		toLog += fmt.Sprintf("%s:%d", p.String(), p.age)
		if idx < len(c.tManView.asArr)-1 {
			toLog += ", "
		}
	}
	c.logger.Info(toLog)
}

func (c *CyclonTMan) sendMessageTmpTransport(msg message.Message, target peer.Peer) {
	c.babel.SendMessageSideStream(msg, target, target.ToUDPAddr(), c.ID(), c.ID())
}
