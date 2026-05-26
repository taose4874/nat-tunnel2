package main

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"net"
	"sync"
	"time"
)

type Client struct {
	Addr     *net.UDPAddr
	LastSeen time.Time
	ClientID string
}

type ServerFramework struct {
	conn      *net.UDPConn
	clients   map[string]*Client
	clientsMu sync.RWMutex
	relayMap  map[string]string
	relayMu   sync.RWMutex
}

type Message struct {
	Path      string          `json:"path"`
	ClientID  string          `json:"clientId"`
	Timestamp int64           `json:"timestamp"`
	Data      json.RawMessage `json:"data,omitempty"`
}

const MAGIC = 0x12345678

func NewServerFramework() (*ServerFramework, error) {
	addr, err := net.ResolveUDPAddr("udp", ":17709")
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	return &ServerFramework{
		conn:     conn,
		clients:  make(map[string]*Client),
		relayMap: make(map[string]string),
	}, nil
}

func (sf *ServerFramework) Start() {
	log.Println("NATUN server starting on :17709")
	go sf.cleanupLoop()
	sf.receiveLoop()
}

func (sf *ServerFramework) Stop() {
	sf.conn.Close()
}

func (sf *ServerFramework) receiveLoop() {
	buf := make([]byte, 65536)
	for {
		n, addr, err := sf.conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Read error: %v", err)
			continue
		}
		go sf.handlePacket(buf[:n], addr)
	}
}

func (sf *ServerFramework) handlePacket(data []byte, addr *net.UDPAddr) {
	if len(data) >= 7 {
		magic := binary.BigEndian.Uint32(data[0:4])
		if magic == MAGIC {
			sf.handleRelayPacket(data, addr)
			return
		}
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err == nil {
		sf.handleMessage(&msg, addr)
	}
}

func (sf *ServerFramework) handleRelayPacket(data []byte, addr *net.UDPAddr) {
	sf.clientsMu.RLock()
	sourceID := ""
	for id, client := range sf.clients {
		if client.Addr.IP.Equal(addr.IP) && client.Addr.Port == addr.Port {
			sourceID = id
			break
		}
	}
	sf.clientsMu.RUnlock()

	if sourceID == "" {
		return
	}

	sf.relayMu.RLock()
	targetID, ok := sf.relayMap[sourceID]
	sf.relayMu.RUnlock()

	if !ok {
		return
	}

	sf.clientsMu.RLock()
	target, ok := sf.clients[targetID]
	sf.clientsMu.RUnlock()

	if ok {
		sf.conn.WriteToUDP(data, target.Addr)
	}
}

func (sf *ServerFramework) handleMessage(msg *Message, addr *net.UDPAddr) {
	switch msg.Path {
	case "register":
		sf.handleRegister(msg, addr)
	case "beat":
		sf.handleBeat(msg, addr)
	case "connectPeer":
		sf.handleConnectPeer(msg, addr)
	}
}

func (sf *ServerFramework) handleRegister(msg *Message, addr *net.UDPAddr) {
	sf.clientsMu.Lock()
	defer sf.clientsMu.Unlock()

	if msg.ClientID != "" {
		sf.clients[msg.ClientID] = &Client{
			Addr:     addr,
			LastSeen: time.Now(),
			ClientID: msg.ClientID,
		}
		log.Printf("Client registered: %s from %v", msg.ClientID, addr)
	}
}

func (sf *ServerFramework) handleBeat(msg *Message, addr *net.UDPAddr) {
	sf.clientsMu.Lock()
	defer sf.clientsMu.Unlock()

	if client, ok := sf.clients[msg.ClientID]; ok {
		client.LastSeen = time.Now()
		client.Addr = addr
	}

	resp := &Message{
		Path:      "beatAck",
		ClientID:  "server",
		Timestamp: msg.Timestamp,
	}
	data, _ := json.Marshal(resp)
	sf.conn.WriteToUDP(data, addr)
}

func (sf *ServerFramework) handleConnectPeer(msg *Message, addr *net.UDPAddr) {
	var data struct {
		TargetClientID string `json:"targetClientId"`
		SourceClientID string `json:"sourceClientId"`
		Password       string `json:"password"`
	}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return
	}

	sf.clientsMu.RLock()
	source, sourceOk := sf.clients[data.SourceClientID]
	target, targetOk := sf.clients[data.TargetClientID]
	sf.clientsMu.RUnlock()

	if !sourceOk || !targetOk {
		log.Printf("Connect failed: client not found (source: %v, target: %v)", sourceOk, targetOk)
		return
	}

	peerInfo := struct {
		ClientID string `json:"clientId"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
	}{
		ClientID: data.TargetClientID,
		Host:     target.Addr.IP.String(),
		Port:     target.Addr.Port,
	}
	peerInfoData, _ := json.Marshal(peerInfo)

	sourceMsg := &Message{
		Path:      "peerInfo",
		ClientID:  "server",
		Timestamp: time.Now().UnixMilli(),
		Data:      peerInfoData,
	}
	sourceMsgData, _ := json.Marshal(sourceMsg)
	sf.conn.WriteToUDP(sourceMsgData, source.Addr)

	connectData, _ := json.Marshal(struct {
		TargetClientID string `json:"targetClientId"`
		Password       string `json:"password"`
	}{
		TargetClientID: data.SourceClientID,
		Password:       data.Password,
	})

	targetMsg := &Message{
		Path:      "connectPeer",
		ClientID:  "server",
		Timestamp: time.Now().UnixMilli(),
		Data:      connectData,
	}
	targetMsgData, _ := json.Marshal(targetMsg)
	sf.conn.WriteToUDP(targetMsgData, target.Addr)

	sf.relayMu.Lock()
	sf.relayMap[data.SourceClientID] = data.TargetClientID
	sf.relayMap[data.TargetClientID] = data.SourceClientID
	sf.relayMu.Unlock()

	log.Printf("Connecting %s to %s", data.SourceClientID, data.TargetClientID)
}

func (sf *ServerFramework) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		sf.clientsMu.Lock()
		now := time.Now()
		for id, client := range sf.clients {
			if now.Sub(client.LastSeen) > 60*time.Second {
				delete(sf.clients, id)
				log.Printf("Client timed out: %s", id)
			}
		}
		sf.clientsMu.Unlock()
	}
}
