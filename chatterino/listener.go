package chatterino

import (
	"bufio"
	"fmt"
	"net"
)

func (srv *ChatServer) sendMessages() {
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

func (srv *ChatServer) readMessages(conn *net.Conn, id string) {
	rdr := bufio.NewReader(*conn)
	for {
		msg, err := rdr.ReadString('\n')
		if err != nil {
			srv.deadConns <- *conn
			break
		}
		srv.messages <- fmt.Sprintf("[%s]: %s", id, msg)
	}
}

func (srv *ChatServer) handleConnections(ln *net.Listener) error {
	go func(ln *net.Listener) {
		for {
			conn, err := (*ln).Accept()
			if err != nil {
				fmt.Printf("couldn't accept new conn: %v\n", err)
				continue
			}
			srv.newConns <- conn
		}
	}(ln)

	for {
		select {
		case conn := <-srv.newConns:
			id := randomSha1(conn.RemoteAddr().String())

			// block mutex to avoid data race on conn & id storing
			srv.mu.Lock()
			srv.clients[conn] = id
			srv.mu.Unlock()

			go srv.readMessages(&conn, id)

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
			return (*ln).Close()
		}
	}
}
