package main

import (
	"flag"
	"fmt"
	"io"
	"time"

	"bytes"
	"io/ioutil"
	"log"
	"net"
	"os"
)

func main() {
	addr := ""
	flag.StringVar(&addr, "address", "localhost:1234", "supply server ip and port")
	flag.Parse()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	const options = `1) Enter the username:
    2) Enter the filename to store:
    3) Enter the filename to retrieve:
    4) Exit:`

	exit := false
	username := ""
	filename := ""

	choice := 0
	for !exit {
		fmt.Println(options)
		fmt.Print("Chose Option: ")
		fmt.Scanf("%d", &choice)
		switch choice {
		case 1:
			fmt.Print("Enter Username: ")
			fmt.Scanf("%s", &username)
		case 2:
			if username == "" {
				fmt.Println()
				fmt.Println("You must enter a username first")
				break
			}
			fmt.Print("Enter Filename: ")
			fmt.Scanf("%s", &filename)
			start := time.Now()
			uploadFile(conn, username, filename)
			duration := time.Since(start)
			fmt.Println("\nDuration: ", duration)
		case 3:
			if username == "" {
				fmt.Println()
				fmt.Println("You must enter a username first")
				break
			}
			fmt.Print("Enter Filename: ")
			fmt.Scanf("%s", &filename)
			start := time.Now()
			downloadFile(conn, username, filename)
			duration := time.Since(start)
			fmt.Println("\nDuration: ", duration)
		case 4:
			if err := conn.Close(); err != nil {
				log.Println(err)
			}
			exit = true
		}
	}
}

func uploadFile(conn io.Writer, username, filename string) error {
	operation := "upload"
	if err := sendMessage(conn, []byte(username)); err != nil {
		return err
	}

	if err := sendMessage(conn, []byte(operation)); err != nil {
		return err
	}

	if err := sendMessage(conn, []byte(filename)); err != nil {
		return err
	}

	fileBuff, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	if err := sendMessage(conn, fileBuff); err != nil {
		return err
	}

	return nil
}

func downloadFile(conn io.ReadWriter, username, filename string) error {
	operation := "download"

	if err := sendMessage(conn, []byte(username)); err != nil {
		return err
	}

	if err := sendMessage(conn, []byte(operation)); err != nil {
		return err
	}

	if err := sendMessage(conn, []byte(filename)); err != nil {
		return err
	}

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	var bufferStatus bytes.Buffer
	if err := readMessage(conn, &bufferStatus); err != nil {
		return err
	}

	log.Println(bufferStatus.String())
	if bufferStatus.String() == "file not found" {
		return nil
	}

	if err := readMessage(conn, f); err != nil {
		return err
	}

	return nil
}
