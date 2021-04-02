package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
)

type FileOwners struct {
	mu        sync.Mutex
	ownership map[string][]string
}

func main() {
	addr := ""
	flag.StringVar(&addr, "address", "localhost:1234", "supply listening ip and port")
	flag.Parse()

	var fo FileOwners
	fo.ownership = make(map[string][]string)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
		}
		go handleUser(conn, &fo)
	}
}

func handleUser(conn net.Conn, fo *FileOwners) {
	for {
		var bufferUserName bytes.Buffer
		if err := readMessage(conn, &bufferUserName); err != nil {
			log.Println(err)
		}

		var bufferOperationName bytes.Buffer
		if err := readMessage(conn, &bufferOperationName); err != nil {
			log.Println(err)
		}

		switch string(bufferOperationName.Bytes()) {
		case "upload":
			var bufferFileName bytes.Buffer
			if err := readMessage(conn, &bufferFileName); err != nil {
				log.Println(err)
			}

			if err := os.MkdirAll(bufferUserName.String(), 0744); err != nil {
				log.Println(err)
			}

			f, err := os.OpenFile(filepath.Join(bufferUserName.String(), bufferFileName.String()), os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Println(err)
			}

			if err := readMessage(conn, f); err != nil {
				log.Println(err)
			}

			fo.ownership[string(bufferUserName.Bytes())] = append(fo.ownership[string(bufferUserName.Bytes())], string(bufferFileName.Bytes()))
			log.Printf("User (%s) uploading file (%s)", bufferUserName.Bytes(), bufferFileName.Bytes())
		case "download":
			var bufferFileName bytes.Buffer
			if err := readMessage(conn, &bufferFileName); err != nil {
				log.Println(err)
			}

			found := false
			for _, filename := range fo.ownership[string(bufferUserName.Bytes())] {
				if filename == bufferFileName.String() {
					found = true
					break
				}
			}

			if !found {
				status := []byte("file not found")
				if err := sendMessage(conn, status); err != nil {
					log.Println(err)
					break
				}
				break
			}

			status := []byte("file found")
			if err := sendMessage(conn, status); err != nil {
				log.Println(err)
			}

			f, err := ioutil.ReadFile(filepath.Join(bufferUserName.String(), bufferFileName.String()))
			if err != nil {
				log.Println(err)
			}

			if err := sendMessage(conn, f); err != nil {
				log.Println(err)
			}

			log.Printf("User (%s) downloading file (%s)", bufferUserName.Bytes(), bufferFileName.Bytes())
		}
	}
	if err := conn.Close(); err != nil {
		log.Println(err)
	}
}
