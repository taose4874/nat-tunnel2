package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
)

type TunDevice struct {
	name string
	file *os.File
}

func NewTunDevice(name string, ip string) (NetDevice, error) {
	cmd := exec.Command("ifconfig", "tun")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	var tunName string
	for _, line := range lines {
		if strings.HasPrefix(line, "tun") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 0 {
				tunName = parts[0]
				break
			}
		}
	}

	if tunName == "" {
		tunName = "tun0"
	}

	devPath := "/dev/" + tunName
	file, err := os.OpenFile(devPath, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	if err := setTunIP(tunName, ip); err != nil {
		file.Close()
		return nil, err
	}

	if err := setTunUp(tunName); err != nil {
		file.Close()
		return nil, err
	}

	return &TunDevice{
		name: tunName,
		file: file,
	}, nil
}

func setTunIP(name string, ip string) error {
	cmd := exec.Command("ifconfig", name, ip, ip)
	return cmd.Run()
}

func setTunUp(name string) error {
	cmd := exec.Command("ifconfig", name, "up")
	return cmd.Run()
}

func (t *TunDevice) Read(packet []byte) (int, error) {
	return t.file.Read(packet)
}

func (t *TunDevice) Write(packet []byte) (int, error) {
	return t.file.Write(packet)
}

func (t *TunDevice) Close() error {
	return t.file.Close()
}
