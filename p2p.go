package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

func handleConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Connection closed.")
			return
		}
		fmt.Println("Received:", strings.TrimSpace(message))
	}
}

func startServer(port string) {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Listening on port:", port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println("Connected to:", conn.RemoteAddr())
		go handleConnection(conn)
	}
}

func startClient(ip, port string) {
	conn, err := net.Dial("tcp", ip+":"+port)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()
	fmt.Println("Connected to:", ip+":"+port)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Send message (or '!exit' to disconnect): ")
		scanner.Scan()
		text := scanner.Text()
		if text == "!exit" {
			break
		}
		conn.Write([]byte(text + "\n"))
	}
}

func main() {
	portPtr := flag.String("p", "8080", "port to listen on")
	flag.Parse()

	go startServer(*portPtr)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Command (or '!exit' to quit): ")
		scanner.Scan()
		cmd := scanner.Text()

		if strings.HasPrefix(cmd, "!connect ") {
			parts := strings.Split(cmd, " ")
			if len(parts) != 2 {
				fmt.Println("Invalid command format. Use '!connect host:port'")
				continue
			}
			addressParts := strings.Split(parts[1], ":")
			if len(addressParts) != 2 {
				fmt.Println("Invalid address format. Use 'host:port'")
				continue
			}
			startClient(addressParts[0], addressParts[1])
		} else if cmd == "!exit" {
			break
		} else {
			fmt.Println("Unknown command.")
		}
	}
}
