package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type Message struct {
	Path      string          `json:"path"`
	ClientID  string          `json:"clientId"`
	Timestamp int64           `json:"timestamp"`
	Data      json.RawMessage `json:"data,omitempty"`
}

type ClientFramework struct {
	conn         *net.UDPConn
	serverAddr   *net.UDPAddr
	handlers     map[string]func(*Message, *net.UDPAddr)
	handlersMu   sync.RWMutex
	relayAddr    *net.UDPAddr
	directAddr   *net.UDPAddr
	receiveChan  chan []byte
	stopChan     chan struct{}
}

const (
	MAGIC       = 0x12345678
	MODE_DIRECT = 0x01
	MODE_RELAY  = 0x02
	BUFFER_SIZE = 1500
)

func NewClientFramework() (*ClientFramework, error) {
	cfg := GetConfig()
	addr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	serverAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port))
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &ClientFramework{
		conn:        conn,
		serverAddr:  serverAddr,
		handlers:    make(map[string]func(*Message, *net.UDPAddr)),
		receiveChan: make(chan []byte, 100),
		stopChan:    make(chan struct{}),
	}, nil
}

func (cf *ClientFramework) Start() {
	go cf.receiveLoop()
	go cf.heartbeatLoop()
	cf.registerToServer()
}

func (cf *ClientFramework) Stop() {
	close(cf.stopChan)
	cf.conn.Close()
}

func (cf *ClientFramework) RegisterHandler(path string, handler func(*Message, *net.UDPAddr)) {
	cf.handlersMu.Lock()
	defer cf.handlersMu.Unlock()
	cf.handlers[path] = handler
}

func (cf *ClientFramework) SendToServer(msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = cf.conn.WriteToUDP(data, cf.serverAddr)
	return err
}

func (cf *ClientFramework) SendTo(addr *net.UDPAddr, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = cf.conn.WriteToUDP(data, addr)
	return err
}

func (cf *ClientFramework) SendDataDirect(data []byte) error {
	if cf.directAddr == nil {
		return fmt.Errorf("no direct address set")
	}
	packet := makePacket(MODE_DIRECT, data)
	_, err := cf.conn.WriteToUDP(packet, cf.directAddr)
	return err
}

func (cf *ClientFramework) SendDataRelay(data []byte) error {
	if cf.relayAddr == nil {
		return fmt.Errorf("no relay address set")
	}
	packet := makePacket(MODE_RELAY, data)
	_, err := cf.conn.WriteToUDP(packet, cf.relayAddr)
	return err
}

func makePacket(mode byte, data []byte) []byte {
	packet := make([]byte, 7+len(data))
	binary.BigEndian.PutUint32(packet[0:4], MAGIC)
	packet[4] = mode
	binary.BigEndian.PutUint16(packet[5:7], uint16(len(data)))
	copy(packet[7:], data)
	return packet
}

func (cf *ClientFramework) SetRelayAddr(addr *net.UDPAddr) {
	cf.relayAddr = addr
}

func (cf *ClientFramework) SetDirectAddr(addr *net.UDPAddr) {
	cf.directAddr = addr
}

func (cf *ClientFramework) GetLocalPort() int {
	return cf.conn.LocalAddr().(*net.UDPAddr).Port
}

func (cf *ClientFramework) receiveLoop() {
	buf := make([]byte, 65536)
	for {
		select {
		case <-cf.stopChan:
			return
		default:
			n, addr, err := cf.conn.ReadFromUDP(buf)
			if err != nil {
				log.Printf("Read error: %v", err)
				continue
			}

			if n >= 7 {
				magic := binary.BigEndian.Uint32(buf[0:4])
				if magic == MAGIC {
					mode := buf[4]
					length := binary.BigEndian.Uint16(buf[5:7])
					if int(length) <= n-7 {
						data := buf[7 : 7+length]
						cf.handleDataPacket(mode, data, addr)
						continue
					}
				}
			}

			var msg Message
			if err := json.Unmarshal(buf[:n], &msg); err == nil {
				cf.handleMessage(&msg, addr)
			}
		}
	}
}

func (cf *ClientFramework) handleMessage(msg *Message, addr *net.UDPAddr) {
	cf.handlersMu.RLock()
	handler, ok := cf.handlers[msg.Path]
	cf.handlersMu.RUnlock()

	if ok {
		go handler(msg, addr)
	}
}

func (cf *ClientFramework) handleDataPacket(mode byte, data []byte, addr *net.UDPAddr) {
	select {
	case cf.receiveChan <- data:
	default:
	}
}

func (cf *ClientFramework) GetReceiveChan() <-chan []byte {
	return cf.receiveChan
}

func (cf *ClientFramework) registerToServer() {
	cfg := GetConfig()
	msg := &Message{
		Path:      "register",
		ClientID:  cfg.ClientID,
		Timestamp: time.Now().UnixMilli(),
	}
	cf.SendToServer(msg)
}

func (cf *ClientFramework) heartbeatLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cf.stopChan:
			return
		case <-ticker.C:
			cfg := GetConfig()
			msg := &Message{
				Path:      "beat",
				ClientID:  cfg.ClientID,
				Timestamp: time.Now().UnixMilli(),
			}
			cf.SendToServer(msg)
		}
	}
}
