package main

import (
	"fmt"
	"math/rand"

	"github.com/nm-morais/go-babel/pkg/peer"
)

type PeerState struct {
	peer.Peer
	age uint16
}

type PeerStateArr []*PeerState

func (s PeerStateArr) Len() int {
	return len(s)
}
func (s PeerStateArr) Less(i, j int) bool {
	return s[i].age > s[j].age
}
func (s PeerStateArr) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s PeerStateArr) ToPeerArr() (toRet []peer.Peer) {
	for _, p := range s {
		toRet = append(toRet, p.Peer)
	}
	return toRet
}

type View struct {
	capacity int
	asArr    PeerStateArr
	asMap    map[string]*PeerState
}

func (v *View) size() int {
	return len(v.asArr)
}

func (v *View) contains(p fmt.Stringer) bool {
	_, ok := v.asMap[p.String()]
	return ok
}

func (v *View) dropRandom() *PeerState {
	toDropIdx := getRandInt(len(v.asArr))
	peerDropped := v.asArr[toDropIdx]
	v.asArr = append(v.asArr[:toDropIdx], v.asArr[toDropIdx+1:]...)
	delete(v.asMap, peerDropped.String())
	return peerDropped
}

func (v *View) add(p *PeerState, dropIfFull bool) {
	if v.isFull() {
		if dropIfFull {
			v.dropRandom()
		} else {
			panic("adding peer to view already full")
		}
	}

	_, alreadyExists := v.asMap[p.String()]
	if !alreadyExists {
		v.asMap[p.String()] = p
		v.asArr = append([]*PeerState{p}, v.asArr...)
	}
}

func (v *View) remove(p peer.Peer) (existed bool) {
	_, existed = v.asMap[p.String()]
	if existed {
		found := false
		for idx, curr := range v.asArr {
			if peer.PeersEqual(curr, p) {
				v.asArr = append(v.asArr[:idx], v.asArr[idx+1:]...)
				found = true
				break
			}
		}
		if !found {
			panic("node was in keys but not in array")
		}
		delete(v.asMap, p.String())
	}
	return existed
}

func (v *View) get(p fmt.Stringer) (*PeerState, bool) {
	elem, exists := v.asMap[p.String()]
	if !exists {
		return nil, false
	}
	return elem, exists
}

func (v *View) isFull() bool {
	return len(v.asArr) >= v.capacity
}

func (v *View) toArray() PeerStateArr {
	return v.asArr
}

func (v *View) getRandomElementsFromView(amount int, exclusions ...peer.Peer) PeerStateArr {
	viewAsArr := v.toArray()
	perm := rand.Perm(len(viewAsArr))
	rndElements := []*PeerState{}
	for i := 0; i < len(viewAsArr) && len(rndElements) < amount; i++ {
		excluded := false
		curr := viewAsArr[perm[i]]
		for _, exclusion := range exclusions {
			if peer.PeersEqual(exclusion, curr) {
				excluded = true
				break
			}
		}
		if !excluded {
			rndElements = append(rndElements, curr)
		}
	}
	return rndElements
}
