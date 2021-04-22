package protocol

import (
	"time"

	"github.com/nm-morais/go-babel/pkg/timer"
)

const GossipTimerID = 2000

type GossipTimer struct {
	duration time.Duration
}

func (GossipTimer) ID() timer.ID {
	return GossipTimerID
}

func (s GossipTimer) Duration() time.Duration {
	return s.duration
}

const ShuffleTimerID = 2001

type ShuffleTimer struct {
	duration time.Duration
}

func (ShuffleTimer) ID() timer.ID {
	return ShuffleTimerID
}

func (s ShuffleTimer) Duration() time.Duration {
	return s.duration
}

const DebugTimerID = 2005

type DebugTimer struct {
	duration time.Duration
}

func (DebugTimer) ID() timer.ID {
	return DebugTimerID
}

func (s DebugTimer) Duration() time.Duration {
	return s.duration
}
