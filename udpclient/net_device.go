package main

type NetDevice interface {
	Read(packet []byte) (int, error)
	Write(packet []byte) (int, error)
	Close() error
}
