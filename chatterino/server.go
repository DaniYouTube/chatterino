package chatterino

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const ChatEOL = "\r\n"

type ChatServer struct {
	// map of clients with their ID
	clients map[net.Conn]string

	newConns  chan net.Conn
	deadConns chan net.Conn
	messages  chan string

	quit chan os.Signal

	mu sync.RWMutex
}

func (srv *ChatServer) Listen(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	fmt.Printf("chatterino started listening on %s!\n", addr)

	go srv.sendMessages()
	return srv.handleConnections(&ln)
}

func NewServer() *ChatServer {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	return &ChatServer{
		clients:   make(map[net.Conn]string),
		newConns:  make(chan net.Conn),
		deadConns: make(chan net.Conn),
		messages:  make(chan string),
		quit:      sigs,
	}
}

func randomSha1(addr string) string {
	h := sha1.New()
	h.Write([]byte(addr))
	rawHash := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return rawHash[:len(rawHash)-1]
}
