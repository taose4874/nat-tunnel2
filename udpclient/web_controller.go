package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

type StatusResponse struct {
	ClientID       string `json:"clientId"`
	Password       string `json:"password"`
	TunIP          string `json:"tunIp"`
	Connected      bool   `json:"connected"`
	ConnectionMode string `json:"connectionMode"`
	RemoteClientID string `json:"remoteClientId,omitempty"`
	RemoteIP       string `json:"remoteIp,omitempty"`
	Latency        int64  `json:"latency,omitempty"`
}

type ConnectRequest struct {
	TargetClientID string `json:"targetClientId"`
	Password       string `json:"password"`
}

func startWebServer() {
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/api/status", handleStatus)
	http.HandleFunc("/api/connect", handleConnect)
	http.HandleFunc("/api/disconnect", handleDisconnect)

	log.Printf("Web server starting on http://127.0.0.1:8898")
	if err := http.ListenAndServe(":8898", nil); err != nil {
		log.Printf("Web server error: %v", err)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(htmlContent))
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	cfg := GetConfig()
	state := GetConnectionState()

	modeStr := "disconnected"
	switch state.GetMode() {
	case ConnectionModeDirect:
		modeStr = "direct"
	case ConnectionModeRelay:
		modeStr = "relay"
	}

	resp := StatusResponse{
		ClientID:       cfg.ClientID,
		Password:       cfg.ClientPwd,
		TunIP:          cfg.TunIP,
		Connected:      state.IsConnected(),
		ConnectionMode: modeStr,
		RemoteClientID: state.GetRemoteClientID(),
		RemoteIP:       state.GetRemoteIP(),
		Latency:        state.GetLatency(),
	}

	json.NewEncoder(w).Encode(resp)
}

func handleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	req.TargetClientID = strings.TrimSpace(req.TargetClientID)
	if req.TargetClientID == "" || req.Password == "" {
		http.Error(w, "Client ID and password required", http.StatusBadRequest)
		return
	}

	cfg := GetConfig()
	msg := &Message{
		Path:      "connectPeer",
		ClientID:  cfg.ClientID,
		Timestamp: time.Now().UnixMilli(),
	}

	msg.Data, _ = json.Marshal(map[string]interface{}{
		"targetClientId": req.TargetClientID,
		"password":       req.Password,
		"sourceClientId": cfg.ClientID,
	})

	clientFramework.SendToServer(msg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state := GetConnectionState()
	state.SetMode(ConnectionModeDisconnected)
	state.SetRemoteClientID("")
	state.SetRemoteIP("")
	state.SetLatency(0)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

const htmlContent = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>NATUN - NAT 穿透组网工具</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            max-width: 500px;
            width: 100%;
            padding: 40px;
        }
        .header {
            text-align: center;
            margin-bottom: 32px;
        }
        .header h1 {
            color: #667eea;
            font-size: 32px;
            margin-bottom: 8px;
        }
        .header p {
            color: #666;
            font-size: 14px;
        }
        .info-card {
            background: #f8f9fa;
            border-radius: 12px;
            padding: 20px;
            margin-bottom: 24px;
        }
        .info-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 12px 0;
            border-bottom: 1px solid #e9ecef;
        }
        .info-item:last-child { border-bottom: none; }
        .info-label { color: #666; font-size: 14px; }
        .info-value {
            color: #333;
            font-weight: 600;
            font-family: monospace;
            font-size: 16px;
            background: white;
            padding: 4px 12px;
            border-radius: 6px;
        }
        .status-badge {
            display: inline-flex;
            align-items: center;
            gap: 6px;
            padding: 6px 12px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: 600;
        }
        .status-badge.connected { background: #d4edda; color: #155724; }
        .status-badge.disconnected { background: #f8d7da; color: #721c24; }
        .status-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
        }
        .status-badge.connected .status-dot {
            background: #28a745;
            animation: pulse 2s infinite;
        }
        .status-badge.disconnected .status-dot { background: #dc3545; }
        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }
        .form-group { margin-bottom: 20px; }
        .form-label {
            display: block;
            color: #333;
            font-weight: 600;
            margin-bottom: 8px;
            font-size: 14px;
        }
        .form-input {
            width: 100%;
            padding: 12px 16px;
            border: 2px solid #e9ecef;
            border-radius: 8px;
            font-size: 16px;
            transition: border-color 0.2s;
        }
        .form-input:focus {
            outline: none;
            border-color: #667eea;
        }
        .btn {
            width: 100%;
            padding: 14px;
            border: none;
            border-radius: 8px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.2s;
        }
        .btn-primary {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        .btn-primary:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
        }
        .btn-danger {
            background: #dc3545;
            color: white;
        }
        .btn-danger:hover { background: #c82333; }
        .connection-info {
            background: #e7f3ff;
            border-radius: 12px;
            padding: 20px;
            margin-bottom: 20px;
        }
        .connection-info h3 {
            color: #0066cc;
            font-size: 16px;
            margin-bottom: 16px;
        }
        .mode-tag {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 4px;
            font-size: 12px;
            font-weight: 600;
        }
        .mode-tag.direct { background: #d4edda; color: #155724; }
        .mode-tag.relay { background: #fff3cd; color: #856404; }
        .hidden { display: none; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🌐 NATUN</h1>
            <p>NAT 穿透组网工具</p>
        </div>
        <div class="info-card">
            <div class="info-item">
                <span class="info-label">连接码 (Client ID)</span>
                <span class="info-value" id="clientId">--</span>
            </div>
            <div class="info-item">
                <span class="info-label">连接密码</span>
                <span class="info-value" id="password">--</span>
            </div>
            <div class="info-item">
                <span class="info-label">虚拟 IP</span>
                <span class="info-value" id="tunIp">--</span>
            </div>
            <div class="info-item">
                <span class="info-label">连接状态</span>
                <span class="status-badge disconnected" id="statusBadge">
                    <span class="status-dot"></span>
                    <span id="statusText">未连接</span>
                </span>
            </div>
        </div>
        <div class="connection-info hidden" id="connectionInfo">
            <h3>连接信息</h3>
            <div class="info-item">
                <span class="info-label">对方设备</span>
                <span class="info-value" id="remoteClientId">--</span>
            </div>
            <div class="info-item">
                <span class="info-label">连接模式</span>
                <span class="mode-tag" id="modeTag">--</span>
            </div>
            <div class="info-item">
                <span class="info-label">延迟</span>
                <span class="info-value" id="latency">-- ms</span>
            </div>
        </div>
        <div id="connectForm">
            <div class="form-group">
                <label class="form-label">目标连接码</label>
                <input type="text" class="form-input" id="targetClientId" placeholder="输入对方的 8 位连接码">
            </div>
            <div class="form-group">
                <label class="form-label">对方连接密码</label>
                <input type="text" class="form-input" id="targetPassword" placeholder="输入对方的连接密码">
            </div>
            <button class="btn btn-primary" onclick="connect()">开始连接</button>
        </div>
        <div class="hidden" id="disconnectForm">
            <button class="btn btn-danger" onclick="disconnect()">断开连接</button>
        </div>
    </div>
    <script>
        async function fetchStatus() {
            try {
                const res = await fetch('/api/status');
                const data = await res.json();
                updateUI(data);
            } catch (e) { console.error(e); }
        }
        function updateUI(data) {
            document.getElementById('clientId').textContent = data.clientId;
            document.getElementById('password').textContent = data.password;
            document.getElementById('tunIp').textContent = data.tunIp;
            const statusBadge = document.getElementById('statusBadge');
            const statusText = document.getElementById('statusText');
            const connectionInfo = document.getElementById('connectionInfo');
            const connectForm = document.getElementById('connectForm');
            const disconnectForm = document.getElementById('disconnectForm');
            if (data.connected) {
                statusBadge.className = 'status-badge connected';
                statusText.textContent = '已连接';
                connectionInfo.classList.remove('hidden');
                connectForm.classList.add('hidden');
                disconnectForm.classList.remove('hidden');
                document.getElementById('remoteClientId').textContent = data.remoteClientId || '--';
                const modeTag = document.getElementById('modeTag');
                modeTag.textContent = data.connectionMode === 'direct' ? '直连 (P2P)' : '中转';
                modeTag.className = 'mode-tag ' + data.connectionMode;
                document.getElementById('latency').textContent = (data.latency || 0) + ' ms';
            } else {
                statusBadge.className = 'status-badge disconnected';
                statusText.textContent = '未连接';
                connectionInfo.classList.add('hidden');
                connectForm.classList.remove('hidden');
                disconnectForm.classList.add('hidden');
            }
        }
        async function connect() {
            const targetClientId = document.getElementById('targetClientId').value.trim();
            const password = document.getElementById('targetPassword').value.trim();
            if (!targetClientId || !password) {
                alert('请输入目标连接码和密码');
                return;
            }
            try {
                await fetch('/api/connect', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ targetClientId, password })
                });
            } catch (e) { console.error(e); }
        }
        async function disconnect() {
            try {
                await fetch('/api/disconnect', { method: 'POST' });
            } catch (e) { console.error(e); }
        }
        fetchStatus();
        setInterval(fetchStatus, 2000);
    </script>
</body>
</html>`
