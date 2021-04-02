package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
)

func main() {
	addr := ""
	flag.StringVar(&addr, "address", "localhost:1234", "supply listening ip and port")
	flag.Parse()

	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		log.Fatal(err)
	}

	node := NewNode(ln.Addr().String())
	rpc.Register(node)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Println(err)
				continue
			}
			rpc.ServeConn(conn)
		}
	}()

	choice := 0
	for {
		fmt.Println("=====================================")
		fmt.Println("Choose one of the following:")
		fmt.Println("1) Enter the peer address to connect")
		fmt.Println("2) Enter the key to find its successor")
		fmt.Println("3) Enter the filename to take its hash")
		fmt.Println("4) Display my-id, succ-id, and pred-id")
		fmt.Println("5) Display the stored file names and their keys")
		fmt.Println("6) Display the finger table")
		fmt.Println("7) Exit")
		fmt.Print("Enter Choice: ")
		fmt.Scanf("%d", &choice)

		switch choice {
		case 1:
			joinaddr := ""
			fmt.Print("Enter Introducer Addr: ")
			fmt.Scanf("%s", &joinaddr)
			node.join(joinaddr, NewRPCCaller())
		case 2:
			var key uint64
			fmt.Print("Enter Key: ")
			fmt.Scanf("%d", &key)
			lr := node.lookup(key, NewRPCCaller())
			fmt.Printf("Found: %v (%v)\n", lr.Addr, lr.ID)
		case 3:
			var filename string
			fmt.Print("Enter Filename: ")
			fmt.Scanf("%d", &filename)
			fmt.Printf("ID: %v\n", ID(filename))
		case 4:
			fmt.Printf("ID (%v)\n", node.id())
			fmt.Printf("Predecessor ID (%v)\n", ID(node.Predecessor))
			fmt.Printf("Successor ID: (%v)\n", ID(node.Successor))
		case 5:
			node.printFileTable()
		case 6:
			node.printFingerTable()
		case 7:
			node.leave(NewRPCCaller())
		}
	}
}
