package chatterino

import (
	"bufio"
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

func (srv *ChatServer) handleMessages() {
	for {
		msg := <-srv.messages
		rawMsg := []byte(msg)

		srv.mu.RLock()
		for conn, id := range srv.clients {
			if _, err := conn.Write(rawMsg); err != nil {
				fmt.Printf("error sending %q to user id %d (%s): %s\n", msg, id, conn.RemoteAddr().String(), err)
			}
		}
		srv.mu.RUnlock()
		fmt.Printf("[CHAT] %s", msg)
	}
}

func (srv *ChatServer) acceptConns(ln *net.Listener) {
	for {
		conn, err := (*ln).Accept()
		if err != nil {
			fmt.Printf("couldn't accept new conn: %v\n", err)
		}
		srv.newConns <- conn
	}
}

func (srv *ChatServer) Listen(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	fmt.Printf("chatterino started listening on %s!\n", addr)

	go srv.acceptConns(&ln)
	go srv.handleMessages()

	for {
		select {
		case conn := <-srv.newConns:
			id := randomSha1(conn.RemoteAddr().String())

			// block mutex to avoid data race on conn & id storing
			srv.mu.Lock()
			srv.clients[conn] = id
			srv.mu.Unlock()

			go func(conn net.Conn, id string) {
				rdr := bufio.NewReader(conn)
				for {
					msg, err := rdr.ReadString('\n')
					if err != nil {
						srv.deadConns <- conn
						break
					}
					srv.messages <- fmt.Sprintf("[%s]: %s", id, msg)
				}
			}(conn, id)

			fmt.Printf("accepted new client %s from %s!\n", id, conn.RemoteAddr().String())
			srv.messages <- fmt.Sprintf("Hit the dabs for %s, who just joined Chatterino! \\o> <o/"+ChatEOL, id)
			break

		case deadConn := <-srv.deadConns:
			id := srv.clients[deadConn]

			fmt.Printf("user %s (%s) disconnected!\n", id, deadConn.RemoteAddr().String())
			srv.messages <- fmt.Sprintf("User %s is no longer with us :("+ChatEOL, id)

			srv.mu.Lock()
			delete(srv.clients, deadConn)
			srv.mu.Unlock()

			break

		case <-srv.quit:
			srv.messages <- "Bye...! Chatterino is being shutdown..." + ChatEOL
			return ln.Close()
		}
	}
}
