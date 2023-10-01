package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

type ClientState struct {
	Conn net.Conn
}

type ServerState struct {
	Conn    net.Conn
	Address string
}

var clients = make(map[string]*ClientState)
var servers = make(map[string]*ServerState)
var lock = sync.Mutex{}

func handleClientConnection(conn net.Conn) {
	defer conn.Close()
	addr := conn.RemoteAddr().String()
	clients[addr] = &ClientState{Conn: conn}
	defer delete(clients, addr)

	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Client", addr, "disconnected.")
			return
		}
		fmt.Println("Received from", addr, ":", strings.TrimSpace(message))
	}
}

func startServer(port string) {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer ln.Close()

	fmt.Println("Listening on port:", port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println("Client connected:", conn.RemoteAddr())
		go handleClientConnection(conn)
	}
}

func main() {
	portPtr := flag.String("p", "8080", "port to listen on")
	flag.Parse()
	go startServer(*portPtr)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Command: ")
		scanner.Scan()
		cmd := scanner.Text()

		switch {
		case strings.HasPrefix(cmd, "!connect "):
			parts := strings.Split(cmd, " ")
			if len(parts) != 2 {
				fmt.Println("Invalid format. Use '!connect host:port'")
				continue
			}
			address := parts[1]
			conn, err := net.Dial("tcp", address)
			if err != nil {
				fmt.Println("Error connecting to server:", err)
				continue
			}
			servers[address] = &ServerState{Conn: conn, Address: address}
			fmt.Println("Connected to server:", address)

		case strings.HasPrefix(cmd, "!send "):
			parts := strings.SplitN(cmd, " ", 3)
			if len(parts) < 3 {
				fmt.Println("Invalid format. Use '!send <server> <message>'")
				continue
			}
			serverAddr, message := parts[1], parts[2]
			server, ok := servers[serverAddr]
			if !ok {
				fmt.Println("No server found with the address", serverAddr)
				continue
			}
			server.Conn.Write([]byte(message + "\n"))

		case strings.HasPrefix(cmd, "!sendall "):
			message := strings.TrimPrefix(cmd, "!sendall ")
			for _, server := range servers {
				server.Conn.Write([]byte(message + "\n"))
			}

		case cmd == "!servers":
			for addr := range servers {
				fmt.Println(addr)
			}

		case cmd == "!clients":
			for addr := range clients {
				fmt.Println(addr)
			}

		case cmd == "!exit":
			for _, server := range servers {
				server.Conn.Close()
			}
			for _, client := range clients {
				client.Conn.Close()
			}
			os.Exit(0)

		default:
			fmt.Println("Unknown command.")
		}
	}
}
