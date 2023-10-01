package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"code.rocketnine.space/tslocum/cview"
	"github.com/gdamore/tcell/v2"
)

type ClientState struct {
	Conn net.Conn
}

type ServerState struct {
	Conn    net.Conn
	Address string
}

// Initialize global state
var clients = make(map[string]*ClientState)
var servers = make(map[string]*ServerState)

// Initialize cview app
var app = cview.NewApplication()
var output = cview.NewTextView()
var input = cview.NewInputField()

func handleClientConnection(conn net.Conn) {
	defer conn.Close()
	addr := conn.RemoteAddr().String()
	clients[addr] = &ClientState{Conn: conn}
	defer delete(clients, addr)

	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			output.SetText(output.GetText(true) + "Client " + addr + " disconnected.\n")
			return
		}
		output.SetText(output.GetText(true) + "Client " + addr + ": " + message)
	}
}

func startServer(port string) {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		output.SetText(output.GetText(true) + "Error starting server: " + err.Error() + "\n")
		return
	}
	defer ln.Close()

	output.SetText(output.GetText(true) + "Listening on port " + port + "\n")
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go handleClientConnection(conn)
	}
}

func processCommand(cmd string) {
	output.SetText(output.GetText(true) + "> " + cmd + "\n")
	switch {
	case strings.HasPrefix(cmd, "!c "):
		parts := strings.Split(cmd, " ")
		if len(parts) != 2 {
			output.SetText(output.GetText(true) + "Invalid format. Use '!connect host:port'\n")
		}
		address := parts[1]
		conn, err := net.Dial("tcp", address)
		if err != nil {
			output.SetText(output.GetText(true) + "Error connecting to server: " + err.Error() + "\n")
		}
		servers[address] = &ServerState{Conn: conn, Address: address}
		output.SetText(output.GetText(true) + "Connected to server " + address + "\n")

	case strings.HasPrefix(cmd, "!s "):
		parts := strings.SplitN(cmd, " ", 3)
		if len(parts) < 3 {
			output.SetText(output.GetText(true) + "Invalid format. Use '!send <server> <message>'\n")
		}
		serverAddr, message := parts[1], parts[2]
		server, ok := servers[serverAddr]
		if !ok {
			output.SetText(output.GetText(true) + "No server found with the address" + serverAddr + "\n")
		}
		server.Conn.Write([]byte(message + "\n"))

	case strings.HasPrefix(cmd, "!sa "):
		message := strings.TrimPrefix(cmd, "!sa ")
		for _, server := range servers {
			server.Conn.Write([]byte(message + "\n"))
		}

	case cmd == "!cs":
		response := "Connected clients:\n"
		for addr := range clients {
			response += addr + "\n"
		}
		output.SetText(output.GetText(true) + response)

	case cmd == "!ss":
		response := "Connected servers:\n"
		for addr, server := range servers {
			response += addr + " (" + server.Address + ")\n"
		}
		output.SetText(output.GetText(true) + response)

	case cmd == "!q":
		for _, server := range servers {
			server.Conn.Close()
		}
		for _, client := range clients {
			client.Conn.Close()
		}
		app.Stop()
		os.Exit(0)

	default:
		output.SetText(output.GetText(true) + "Unknown command: " + cmd + "\n")
	}

	output.ScrollToEnd()
}

func CLI() {
	flex := cview.NewFlex()
	flex.SetDirection(cview.FlexRow)
	flex.AddItem(output, 0, 1, false)
	flex.AddItem(input, 1, 0, true)

	app.SetRoot(flex, true)
	app.SetFocus(input)

	output.SetText("Enter commands below:\n")
	output.SetDynamicColors(true)
	output.SetScrollable(true)
	output.SetChangedFunc(func() {
		app.Draw()
	})

	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			cmd := input.GetText()
			input.SetText("")

			// TODO: make processComand return string not print so we can print to output
			processCommand(cmd)

			app.SetFocus(input)
		}
	})

	// Start the application event loop
	app.Run()
}

func main() {
	portPtr := flag.String("p", "8080", "port to listen on")
	flag.Parse()

	go startServer(*portPtr)

	CLI()
}
