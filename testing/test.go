package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"net"
)

func checkError(err error) {
	if err != nil {
		fmt.Println("Error: %s", err.Error())
		os.Exit(1)
	}
}

func readAll(filePth string) ([]byte, error) {
	f, err := os.Open(filePth)
	checkError(err)
	return ioutil.ReadAll(f)
}

func send(filePth string) {	
	conn, err := net.Dial("tcp", "127.0.0.1:12345")
	checkError(err)
	defer conn.Close()
	text, err := readAll(filePth)
	checkError(err)
	fmt.Println(string(text))
	conn.Write(text)
	fmt.Println("send msg")
}


func main() {
	args := os.Args
    if args == nil || len(args) != 2{
        return
    }
    path := args[1]
    send(path)
}
