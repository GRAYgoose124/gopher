# P2P Chat in Go

Learning Golang. kthxbai.

## Usage
1. Run the first peer with the following command:
```bash
# go run p2p.go [-p listen-port] 
go run p2p.go -p 5000

# after the other peer is running, connect to it
> !c localhost:5001
> !sa hi
```

2. Run the second peer with the following command:
```bash
# in another terminal
go run p2p.go -p 5001
> !c localhost:5000
> !sa hello
```

Use `!h` to see the list of available commands.