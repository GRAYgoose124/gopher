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
	Conn          net.Conn
	ListeningPort string
	BackConnected bool
}

type ServerState struct {
	Conn    net.Conn
	Address string
}

type PeerApp struct {
	App            *cview.Application
	Output         *cview.TextView
	Input          *cview.InputField
	Clients        map[string]*ClientState
	Servers        map[string]*ServerState
	ListeningPort  string
	CommandHistory []string
	CurrentCmdIdx  int
}

func NewPeerApp() *PeerApp {
	app := cview.NewApplication()
	output := cview.NewTextView()
	input := cview.NewInputField()

	return &PeerApp{
		App:            app,
		Output:         output,
		Input:          input,
		Clients:        make(map[string]*ClientState),
		Servers:        make(map[string]*ServerState),
		CommandHistory: []string{},
		CurrentCmdIdx:  0,
	}
}

func (p *PeerApp) handleClientConnection(conn net.Conn) {
	defer conn.Close()
	addr := conn.RemoteAddr().String()
	clientState := &ClientState{Conn: conn, BackConnected: false}
	p.Clients[addr] = clientState
	defer delete(p.Clients, addr)

	reader := bufio.NewReader(conn)
	initialMessage, err := reader.ReadString('\n')
	if err != nil {
		p.App.QueueUpdateDraw(func() {
			p.Output.SetText(p.Output.GetText(true) + "Error reading from client " + addr + ": " + err.Error() + "\n")
		})
		return
	}

	// Check if the initial message is in the format "LISTENING_ON:<port>"
	if strings.HasPrefix(initialMessage, "LISTENING_ON:") {
		clientState.ListeningPort = strings.TrimSpace(strings.TrimPrefix(initialMessage, "LISTENING_ON:"))
		if !clientState.BackConnected && clientState.ListeningPort != "" {
			go func() {
				backConn, err := net.Dial("tcp", addr[:strings.LastIndex(addr, ":")]+":"+clientState.ListeningPort)
				if err != nil {
					p.App.QueueUpdateDraw(func() {
						p.Output.SetText(p.Output.GetText(true) + "Error creating back connection to client: " + err.Error() + "\n")
					})
					return
				}
				serverAddr := backConn.RemoteAddr().String()
				p.Servers[serverAddr] = &ServerState{Conn: backConn, Address: serverAddr}
				clientState.BackConnected = true
			}()
		}
	} else {
		p.App.QueueUpdateDraw(func() {
			p.Output.SetText(p.Output.GetText(true) + "Client " + addr + " did not send a valid listening port.\n")
		})
		return
	}

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			p.App.QueueUpdateDraw(func() {
				p.Output.SetText(p.Output.GetText(true) + "Client " + addr + " disconnected.\n")
			})
			return
		}
		p.App.QueueUpdateDraw(func() {
			p.Output.SetText(p.Output.GetText(true) + "Client " + addr + ": " + message)
		})
	}
}

func (p *PeerApp) startServer(port string) {
	ln, err := net.Listen("tcp", ":"+port)
	p.ListeningPort = port
	if err != nil {
		p.App.QueueUpdateDraw(func() {
			p.Output.SetText(p.Output.GetText(true) + "Error starting server: " + err.Error() + "\n")
		})
		return
	}
	defer ln.Close()

	p.App.QueueUpdateDraw(func() {
		p.Output.SetText(p.Output.GetText(true) + "Listening on port " + port + "\n")
	})
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go p.handleClientConnection(conn)
	}
}

func (p *PeerApp) processCommand(cmd string) {
	p.Output.SetText(p.Output.GetText(true) + "> " + cmd + "\n")

	p.CommandHistory = append(p.CommandHistory, cmd)
	p.CurrentCmdIdx = len(p.CommandHistory)

	switch {
	case strings.HasPrefix(cmd, "!c "):
		parts := strings.Split(cmd, " ")
		if len(parts) != 2 {
			p.App.QueueUpdateDraw(func() {
				p.Output.SetText(p.Output.GetText(true) + "Invalid format. Use '!connect host:port'\n")
			})
			return
		}
		address := parts[1]
		conn, err := net.Dial("tcp", address)
		if err != nil {
			p.App.QueueUpdateDraw(func() {
				p.Output.SetText(p.Output.GetText(true) + "Error connecting to server: " + err.Error() + "\n")
			})
			return
		}

		// Send the LISTENING_ON message with the client's listening port
		fmt.Fprintf(conn, "LISTENING_ON:%s\n", p.ListeningPort)

		p.Servers[address] = &ServerState{Conn: conn, Address: address}
		p.App.QueueUpdateDraw(func() {
			p.Output.SetText(p.Output.GetText(true) + "Connected to server " + address + "\n")
		})

	case strings.HasPrefix(cmd, "!s "):
		parts := strings.SplitN(cmd, " ", 3)
		if len(parts) < 3 {
			p.App.QueueUpdateDraw(func() {
				p.Output.SetText(p.Output.GetText(true) + "Invalid format. Use '!send <server> <message>'\n")
			})
			return
		}
		serverAddr, message := parts[1], parts[2]
		server, ok := p.Servers[serverAddr]
		if !ok {
			p.App.QueueUpdateDraw(func() {
				p.Output.SetText(p.Output.GetText(true) + "Not connected to server " + serverAddr + "\n")
			})
			return
		}
		server.Conn.Write([]byte(message + "\n"))

	case strings.HasPrefix(cmd, "!sa "):
		message := strings.TrimPrefix(cmd, "!sa ")
		for _, server := range p.Servers {
			server.Conn.Write([]byte(message + "\n"))
		}

	case cmd == "!cs":
		response := "Connected clients:\n"
		for addr := range p.Clients {
			response += addr + "\n"
		}
		p.App.QueueUpdateDraw(func() {
			p.Output.SetText(p.Output.GetText(true) + response)
		})

	case cmd == "!ss":
		response := "Connected servers:\n"
		for addr, server := range p.Servers {
			response += addr + " (" + server.Address + ")\n"
		}
		p.App.QueueUpdateDraw(func() {
			p.Output.SetText(p.Output.GetText(true) + response)
		})

	case cmd == "!q":
		for _, server := range p.Servers {
			server.Conn.Close()
		}
		for _, client := range p.Clients {
			client.Conn.Close()
		}
		p.App.Stop()
		os.Exit(0)

	case cmd == "!h":
		p.App.QueueUpdateDraw(func() {
			p.Output.SetText(p.Output.GetText(true) + "Commands:\n" +
				"\t!c <host:port> - connect to a server\n" +
				"\t!s <server> <message> - send a message to a server\n" +
				"\t!sa <message> - send a message to all servers\n" +
				"\t!cs - list connected clients\n" +
				"\t!ss - list connected servers\n" +
				"\t!q - quit\n")
		})
	default:
		p.App.QueueUpdateDraw(func() {
			p.Output.SetText(p.Output.GetText(true) + "Unknown command: " + cmd + "\n")
		})
	}

	p.Output.ScrollToEnd()
}

func (p *PeerApp) CLI() {
	flex := cview.NewFlex()
	flex.SetDirection(cview.FlexRow)
	flex.AddItem(p.Output, 0, 1, false)
	flex.AddItem(p.Input, 1, 0, true)

	p.App.SetRoot(flex, true)
	p.App.SetFocus(p.Input)

	p.Output.SetText(" -- Welcome to Gopher --\n")
	p.Output.SetDynamicColors(true)
	p.Output.SetScrollable(true)
	p.Output.SetChangedFunc(func() {
		p.App.Draw()
	})

	p.Input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			cmd := p.Input.GetText()
			p.Input.SetText("")

			p.processCommand(cmd)
			p.App.SetFocus(p.Input)
		case tcell.KeyUp:
			if len(p.CommandHistory) == 0 {
				return
			}
			if p.CurrentCmdIdx > 0 {
				p.CurrentCmdIdx--
			}
			p.Input.SetText(p.CommandHistory[p.CurrentCmdIdx])
		case tcell.KeyDown:
			if p.CurrentCmdIdx < len(p.CommandHistory)-1 {
				p.CurrentCmdIdx++
				p.Input.SetText(p.CommandHistory[p.CurrentCmdIdx])
			} else {
				p.Input.SetText("")
				p.CurrentCmdIdx = len(p.CommandHistory)
			}
		}
	})

	p.App.Run()
}

func main() {
	portPtr := flag.String("p", "8080", "port to listen on")
	flag.Parse()

	peerApp := NewPeerApp()

	go peerApp.startServer(*portPtr)

	peerApp.CLI()
}
