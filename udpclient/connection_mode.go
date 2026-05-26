package main

import "sync"

type ConnectionMode int

const (
	ConnectionModeDisconnected ConnectionMode = iota
	ConnectionModeDirect
	ConnectionModeRelay
)

type ConnectionState struct {
	mode           ConnectionMode
	remoteClientID string
	remoteIP       string
	latency        int64
	mu             sync.RWMutex
}

var connectionState = &ConnectionState{
	mode: ConnectionModeDisconnected,
}

func GetConnectionState() *ConnectionState {
	return connectionState
}

func (s *ConnectionState) GetMode() ConnectionMode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mode
}

func (s *ConnectionState) SetMode(mode ConnectionMode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mode = mode
}

func (s *ConnectionState) GetRemoteClientID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.remoteClientID
}

func (s *ConnectionState) SetRemoteClientID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.remoteClientID = id
}

func (s *ConnectionState) GetRemoteIP() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.remoteIP
}

func (s *ConnectionState) SetRemoteIP(ip string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.remoteIP = ip
}

func (s *ConnectionState) GetLatency() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.latency
}

func (s *ConnectionState) SetLatency(latency int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latency = latency
}

func (s *ConnectionState) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mode == ConnectionModeDirect || s.mode == ConnectionModeRelay
}
