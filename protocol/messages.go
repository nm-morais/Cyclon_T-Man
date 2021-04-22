package protocol

import (
	"encoding/binary"

	"github.com/nm-morais/go-babel/pkg/message"
	"github.com/nm-morais/go-babel/pkg/peer"
)

const ShuffleMessageType = 2000

type ShuffleMessage struct {
	peers []peer.Peer
	ages  []uint16
}
type shuffleMessageSerializer struct{}

func NewShuffleMsg(peers []*PeerState) *ShuffleMessage {

	peerArr := make([]peer.Peer, len(peers))
	ages := make([]uint16, len(peers))
	for idx, p := range peers {
		peerArr[idx] = p.Peer
		ages[idx] = uint16(p.age)
	}

	return &ShuffleMessage{
		peers: peerArr,
		ages:  ages,
	}
}

var defaultShuffleMsgSerializer = shuffleMessageSerializer{}

func (*ShuffleMessage) Type() message.ID                   { return ShuffleMessageType }
func (*ShuffleMessage) Serializer() message.Serializer     { return defaultShuffleMsgSerializer }
func (*ShuffleMessage) Deserializer() message.Deserializer { return defaultShuffleMsgSerializer }
func (shuffleMessageSerializer) Serialize(msg message.Message) []byte {
	shuffleMsg := msg.(*ShuffleMessage)
	peersBytes := peer.SerializePeerArray(shuffleMsg.peers)
	ageBytes := make([]byte, 2*len(shuffleMsg.peers))
	curr := 0
	for _, v := range shuffleMsg.ages {
		binary.BigEndian.PutUint16(ageBytes[curr:], v)
		curr += 2
	}
	return append(peersBytes, ageBytes...)
}
func (shuffleMessageSerializer) Deserialize(msgBytes []byte) message.Message {
	curr, peers := peer.DeserializePeerArray(msgBytes)
	ages := make([]uint16, len(peers))
	for i := 0; i < len(peers); i++ {
		ages[i] = binary.BigEndian.Uint16(msgBytes[curr:])
		curr += 2
	}
	return &ShuffleMessage{
		peers: peers,
		ages:  ages,
	}
}

func (sr ShuffleMessage) ToPeerStateArr() (peers []*PeerState) {
	peers = make([]*PeerState, len(sr.peers))
	for i, p := range sr.peers {
		peers[i] = &PeerState{
			Peer: p,
			age:  sr.ages[i],
		}
	}
	return peers
}

const ShuffleMessageReplyType = 2001

type ShuffleMessageReply struct {
	peers []peer.Peer
	ages  []uint16
}

type shuffleMessageReplySerializer struct{}

func NewShuffleMsgReply(peers []*PeerState) *ShuffleMessageReply {
	peerArr := make([]peer.Peer, len(peers))
	ages := make([]uint16, len(peers))
	for idx, p := range peers {
		peerArr[idx] = p.Peer
		ages[idx] = uint16(p.age)
	}

	return &ShuffleMessageReply{
		peers: peerArr,
		ages:  ages,
	}
}

var defaultShuffleMsgReplySerializer = shuffleMessageReplySerializer{}

func (*ShuffleMessageReply) Type() message.ID               { return ShuffleMessageReplyType }
func (*ShuffleMessageReply) Serializer() message.Serializer { return defaultShuffleMsgReplySerializer }
func (*ShuffleMessageReply) Deserializer() message.Deserializer {
	return defaultShuffleMsgReplySerializer
}
func (shuffleMessageReplySerializer) Serialize(msg message.Message) []byte {
	shuffleMsg := msg.(*ShuffleMessageReply)
	peersBytes := peer.SerializePeerArray(shuffleMsg.peers)
	ageBytes := make([]byte, 2*len(shuffleMsg.peers))
	curr := 0
	for _, v := range shuffleMsg.ages {
		binary.BigEndian.PutUint16(ageBytes[curr:], v)
		curr += 2
	}
	return append(peersBytes, ageBytes...)
}
func (shuffleMessageReplySerializer) Deserialize(msgBytes []byte) message.Message {
	curr, peers := peer.DeserializePeerArray(msgBytes)
	ages := make([]uint16, len(peers))
	for i := 0; i < len(peers); i++ {
		ages[i] = binary.BigEndian.Uint16(msgBytes[curr:])
		curr += 2
	}
	return &ShuffleMessageReply{
		peers: peers,
		ages:  ages,
	}
}

func (sr ShuffleMessageReply) ToPeerStateArr() (peers []*PeerState) {
	peers = make([]*PeerState, len(sr.peers))
	for i, p := range sr.peers {
		peers[i] = &PeerState{
			Peer: p,
			age:  sr.ages[i],
		}
	}
	return peers
}

const TManGossipMsgType = 2002

type TManGossipMsg struct {
	peers []peer.Peer
}
type TManGossipMsgSerializer struct{}

func NewTManGossipMsg(peers []peer.Peer) *TManGossipMsg {
	return &TManGossipMsg{
		peers: peers,
	}
}

var tManGossipMsgSerializer = &TManGossipMsgSerializer{}

func (*TManGossipMsg) Type() message.ID                   { return TManGossipMsgType }
func (*TManGossipMsg) Serializer() message.Serializer     { return tManGossipMsgSerializer }
func (*TManGossipMsg) Deserializer() message.Deserializer { return tManGossipMsgSerializer }
func (*TManGossipMsgSerializer) Serialize(msg message.Message) []byte {
	shuffleMsg := msg.(*TManGossipMsg)
	peersBytes := peer.SerializePeerArray(shuffleMsg.peers)
	ageBytes := make([]byte, 2*len(shuffleMsg.peers))
	return append(peersBytes, ageBytes...)
}
func (TManGossipMsgSerializer) Deserialize(msgBytes []byte) message.Message {
	_, peers := peer.DeserializePeerArray(msgBytes)
	return &TManGossipMsg{
		peers: peers,
	}
}

const tManGossipMsgReplyType = 2003

type tManGossipMsgReply struct {
	peers []peer.Peer
}

type tManGossipMsgReplySerializer struct{}

func NewTManGossipMsgReply(peers []*PeerState) tManGossipMsgReply {
	peerArr := make([]peer.Peer, len(peers))
	for idx, p := range peers {
		peerArr[idx] = p.Peer
	}

	return tManGossipMsgReply{
		peers: peerArr,
	}
}

var defaultGossipMsgSerializer = tManGossipMsgReplySerializer{}

func (tManGossipMsgReply) Type() message.ID { return tManGossipMsgReplyType }
func (tManGossipMsgReply) Serializer() message.Serializer {
	return defaultGossipMsgSerializer
}
func (tManGossipMsgReply) Deserializer() message.Deserializer {
	return defaultGossipMsgSerializer
}
func (tManGossipMsgReplySerializer) Serialize(msg message.Message) []byte {
	gossipMsg := msg.(tManGossipMsgReply)
	peersBytes := peer.SerializePeerArray(gossipMsg.peers)
	return peersBytes
}
func (tManGossipMsgReplySerializer) Deserialize(msgBytes []byte) message.Message {
	_, peers := peer.DeserializePeerArray(msgBytes)
	return tManGossipMsgReply{
		peers: peers,
	}
}
