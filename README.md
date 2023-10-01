# P2P Chat in Go

Learning Golang. kthxbai.

## Building
```bash
go build ./cmd/p2p
```

## Usage
1. Run the first peer with the following command:
```bash
# ./p2p [-p listen-port] 
go run ./cmd/p2p -p 5000

# after the other peer is running, connect to it
> !c localhost:5001
> !sa hi
```

2. Run the second peer in another terminal with the following command:
```bash
go run ./cmd/p2p -p 5001
> !c localhost:5000
> !sa hello
```

Use `!h` to see the list of available commands.