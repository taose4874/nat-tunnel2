package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
)

var (
	clientFramework *ClientFramework
	tunDevice       NetDevice
	punchingMu      sync.Mutex
	punchingDone    chan struct{}
)

type ConnectPeerData struct {
	TargetClientID string `json:"targetClientId"`
	Password       string `json:"password"`
}

type PeerInfo struct {
	ClientID string `json:"clientId"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
}

func main() {
	cfg := GetConfig()
	log.Printf("Starting NATUN client. ClientID: %s, Password: %s", cfg.ClientID, cfg.ClientPwd)

	var err error
	clientFramework, err = NewClientFramework()
	if err != nil {
		log.Fatalf("Failed to create client framework: %v", err)
	}
	defer clientFramework.Stop()

	go startWebServer()

	clientFramework.RegisterHandler("ping", handlePing)
	clientFramework.RegisterHandler("pong", handlePong)
	clientFramework.RegisterHandler("connectPeer", handleConnectPeer)
	clientFramework.RegisterHandler("beatAck", handleBeatAck)
	clientFramework.RegisterHandler("peerInfo", handlePeerInfo)

	clientFramework.Start()

	go setupTunDevice()

	go dataForwardLoop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
}

func setupTunDevice() {
	cfg := GetConfig()
	var err error

	switch runtime.GOOS {
	case "linux":
		tunDevice, err = NewTunDevice("tun0", cfg.TunIP)
	case "darwin":
		tunDevice, err = NewTunDevice("tun0", cfg.TunIP)
	case "windows":
		tunDevice, err = NewTunDevice("", cfg.TunIP)
	default:
		log.Printf("Unsupported OS: %s", runtime.GOOS)
		return
	}

	if err != nil {
		log.Printf("Failed to create TUN device: %v", err)
		return
	}
	defer tunDevice.Close()

	log.Printf("TUN device created with IP: %s", cfg.TunIP)

	buf := make([]byte, 1500)
	for {
		n, err := tunDevice.Read(buf)
		if err != nil {
			log.Printf("TUN read error: %v", err)
			break
		}

		state := GetConnectionState()
		if !state.IsConnected() {
			continue
		}

		if state.GetMode() == ConnectionModeDirect {
			clientFramework.SendDataDirect(buf[:n])
		} else if state.GetMode() == ConnectionModeRelay {
			clientFramework.SendDataRelay(buf[:n])
		}
	}
}

func dataForwardLoop() {
	for data := range clientFramework.GetReceiveChan() {
		if tunDevice == nil {
			continue
		}
		tunDevice.Write(data)
	}
}

func handlePing(msg *Message, addr *net.UDPAddr) {
	cfg := GetConfig()
	response := &Message{
		Path:      "pong",
		ClientID:  cfg.ClientID,
		Timestamp: time.Now().UnixMilli(),
		Data:      msg.Data,
	}
	clientFramework.SendTo(addr, response)
}

func handlePong(msg *Message, addr *net.UDPAddr) {
	punchingMu.Lock()
	if punchingDone != nil {
		close(punchingDone)
		punchingDone = nil
	}
	punchingMu.Unlock()

	clientFramework.SetDirectAddr(addr)
	GetConnectionState().SetMode(ConnectionModeDirect)
	GetConnectionState().SetRemoteClientID(msg.ClientID)
	log.Printf("Direct connection established with %s", msg.ClientID)
}

func handleConnectPeer(msg *Message, addr *net.UDPAddr) {
	var data ConnectPeerData
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		log.Printf("Failed to unmarshal connectPeer data: %v", err)
		return
	}

	cfg := GetConfig()
	if data.Password != cfg.ClientPwd {
		log.Printf("Invalid password from %s", data.TargetClientID)
		return
	}

	go func() {
		time.Sleep(500 * time.Millisecond)
		startPunching(addr, data.TargetClientID)
	}()
}

func handleBeatAck(msg *Message, addr *net.UDPAddr) {
	latency := time.Now().UnixMilli() - msg.Timestamp
	GetConnectionState().SetLatency(latency)
}

func handlePeerInfo(msg *Message, addr *net.UDPAddr) {
	var peerInfo PeerInfo
	if err := json.Unmarshal(msg.Data, &peerInfo); err != nil {
		log.Printf("Failed to unmarshal peerInfo data: %v", err)
		return
	}

	targetAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", peerInfo.Host, peerInfo.Port))
	if err != nil {
		log.Printf("Failed to resolve peer address: %v", err)
		return
	}

	clientFramework.SetRelayAddr(addr)
	GetConnectionState().SetRemoteClientID(peerInfo.ClientID)
	GetConnectionState().SetRemoteIP(fmt.Sprintf("%s:%d", peerInfo.Host, peerInfo.Port))

	go func() {
		if !startPunching(targetAddr, peerInfo.ClientID) {
			cfg := GetConfig()
			if cfg.PunchHole.EnableRelay && cfg.PunchHole.RelayFallback {
				log.Printf("Punching failed, switching to relay mode")
				GetConnectionState().SetMode(ConnectionModeRelay)
			}
		}
	}()
}

func startPunching(targetAddr *net.UDPAddr, targetClientID string) bool {
	punchingMu.Lock()
	if punchingDone != nil {
		punchingMu.Unlock()
		return false
	}
	punchingDone = make(chan struct{})
	punchingMu.Unlock()

	cfg := GetConfig()
	maxConcurrency := cfg.PunchHole.MaxConcurrency
	portRange := cfg.PunchHole.PortRange
	timeout := time.Duration(cfg.PunchHole.PunchTimeout) * time.Second

	basePort := targetAddr.Port - cfg.PunchHole.BasePortOffset
	ports := generatePortRange(basePort, portRange)
	shufflePorts(ports)

	log.Printf("Starting NAT punching to %s, scanning %d ports with concurrency %d", targetClientID, len(ports), maxConcurrency)

	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	go func() {
		for _, port := range ports {
			select {
			case <-punchingDone:
				return
			default:
			}

			semaphore <- struct{}{}
			wg.Add(1)

			go func(p int) {
				defer func() {
					<-semaphore
					wg.Done()
				}()

				addr := &net.UDPAddr{
					IP:   targetAddr.IP,
					Port: p,
				}

				for round := 0; round < 3; round++ {
					select {
					case <-punchingDone:
						return
					default:
					}

					cfg := GetConfig()
					msg := &Message{
						Path:      "ping",
						ClientID:  cfg.ClientID,
						Timestamp: time.Now().UnixMilli(),
					}
					clientFramework.SendTo(addr, msg)
					time.Sleep(300 * time.Millisecond)
				}
			}(port)
		}
	}()

	doneChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneChan)
	}()

	select {
	case <-punchingDone:
		return true
	case <-time.After(timeout):
		punchingMu.Lock()
		if punchingDone != nil {
			close(punchingDone)
			punchingDone = nil
		}
		punchingMu.Unlock()
		return false
	case <-doneChan:
		return false
	}
}

func generatePortRange(basePort, count int) []int {
	ports := make([]int, 0, count)
	for i := 0; i < count; i++ {
		port := basePort + i
		if port > 0 && port < 65536 {
			ports = append(ports, port)
		}
	}
	return ports
}

func shufflePorts(ports []int) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := len(ports) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		ports[i], ports[j] = ports[j], ports[i]
	}
}
