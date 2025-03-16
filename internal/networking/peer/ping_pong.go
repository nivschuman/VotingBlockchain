package networking_peer

import "time"

type PingPongDetails struct {
	Nonce    uint64        //nonce of outgoing ping, 0 if no outgoing ping
	PingTime time.Time     //time of which last ping was sent
	PongTime time.Time     //time of which last pong was returned
	Latency  time.Duration //latency of this peer, PongTime - PingTime
}
