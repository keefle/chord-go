package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"
)

func main() {

	nodeAddr := ""
	fmt.Print("Enter a peer node address: ")
	fmt.Scanf("%s\n", &nodeAddr)

	for {
		fmt.Println("1) Enter the filename to store:")
		fmt.Println("2) Enter the filename to retrieve:")
		fmt.Println("3) Exit")

		choice := 0
		fmt.Print("Enter choice: ")
		fmt.Scanf("%d\n", &choice)
		switch choice {
		case 1:
			fmt.Print("Enter filename to store: ")
			filename := ""
			fmt.Scanf("%s\n", &filename)
			start := time.Now()
			UploadFile(filename, nodeAddr)
			duration := time.Since(start)
			fmt.Println("\nDuration: ", duration)
		case 2:
			fmt.Print("Enter filename to retrieve: ")
			filename := ""
			fmt.Scanf("%s\n", &filename)
			start := time.Now()
			RetrieveFile(filename, nodeAddr)
			duration := time.Since(start)
			fmt.Println("\nDuration: ", duration)
		case 3:
			fmt.Print("Exiting")
			return
		}
	}
}

func UploadFile(filename, nodeAddr string) {
	rpccaller := NewRPCCaller()

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
		return
	}

	var lr LookupResp
	rpccaller.Call(nodeAddr, "Lookup", ID(filename), &lr)

	var ufr UploadFileResp
	rpccaller.Call(lr.Addr, "UploadFile", UploadFileReq{
		Filename: filename,
		Content:  content,
		ID:       ID(filename),
	}, &ufr)
}

func RetrieveFile(filename, nodeAddr string) {
	rpccaller := NewRPCCaller()

	var lr LookupResp
	rpccaller.Call(nodeAddr, "Lookup", ID(filename), &lr)

	var rfr RetrieveFileResp
	rpccaller.Call(lr.Addr, "RetrieveFile", RetrieveFileReq{
		Filename: filename,
		ID:       ID(filename),
	}, &rfr)

	f, err := os.OpenFile(rfr.Filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
		return
	}

	if _, err := f.Write(rfr.Content); err != nil {
		log.Println(err)
		return
	}
}
