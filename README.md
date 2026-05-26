# NATUN - NAT 穿透组网工具

基于 TUN 虚拟网卡技术的 NAT 穿透组网解决方案，让远程设备轻松建立虚拟局域网连接。

## 功能特性

- 🔧 高效 NAT 穿透：并发端口扫描 + 随机化策略，穿透成功率高
- ⚡ 双模式连接：直连模式（P2P）优先，中转模式自动回退
- 🏢 私有部署支持：支持自建中转服务器，满足企业级需求
- 🌍 跨平台支持：Windows、Linux、macOS 全平台兼容
- 🎨 现代 Web 界面：直观的设备连接管理
- 📊 实时监控：连接状态、延迟监控一目了然

## 快速开始

### 客户端

```bash
cd udpclient
go mod tidy
go build
sudo ./natun  # Linux/macOS 需要管理员/root 权限
```

启动后访问 http://127.0.0.1:8898

### 服务器

```bash
cd udpcloud
go mod tidy
go build
./server
```

服务器默认监听 17709 端口。

## 使用说明

1. 在两台设备上分别启动客户端
2. 访问 Web 界面查看各自的连接码和密码
3. 在任意一台设备上输入对方的连接码和密码
4. 点击连接，等待建立连接
5. 连接成功后，可以通过虚拟 IP 互相访问

## 配置

配置文件自动生成在用户目录下的 `.natun/config.json`：

```json
{
  "punch_hole": {
    "max_concurrency": 2,
    "port_range": 16,
    "base_port_offset": 0,
    "enable_relay": true,
    "punch_timeout": 20,
    "relay_fallback": true
  },
  "server": {
    "host": "your-server-ip",
    "port": 17709
  },
  "log_level": "INFO",
  "tun_ip": "10.10.10.199",
  "client_id": "12345678",
  "client_pwd": "123456"
}
```

## 项目结构

```
natun/
├── udpclient/      # 客户端代码
└── udpcloud/       # 服务器代码
```

## 构建

### 客户端

```bash
cd udpclient
chmod +x build.sh
./build.sh
```

### 服务器

```bash
cd udpcloud
chmod +x build.sh
./build.sh
```

## License

MIT License
