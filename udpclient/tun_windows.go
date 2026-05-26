package main

import (
	"fmt"
	"net"
)

type TunDevice struct {
	name string
}

func NewTunDevice(name string, ip string) (NetDevice, error) {
	return &TunDevice{name: name}, fmt.Errorf("Windows TUN device not implemented")
}

func (t *TunDevice) Read(packet []byte) (int, error) {
	return 0, nil
}

func (t *TunDevice) Write(packet []byte) (int, error) {
	return 0, nil
}

func (t *TunDevice) Close() error {
	return nil
}
