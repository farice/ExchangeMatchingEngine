package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"net"
	"time"
	"math/rand"
	"strconv"
	"sync"
)

var accountId = 100001
const letterBytes = "ABC"

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

func sendString(content string) {
	conn, err := net.Dial("tcp", "127.0.0.1:12345")
	checkError(err)
	defer conn.Close()
	//fmt.Println(string(text))
	conn.Write([]byte(content))
	//fmt.Println("send msg")
}

func send(filePth string) {	
	conn, err := net.Dial("tcp", "127.0.0.1:12345")
	checkError(err)
	defer conn.Close()
	text, err := readAll(filePth)
	checkError(err)
	fmt.Println(string(text))
	conn.Write(text)
	//fmt.Println("send msg")
}

func RandStringBytes(n int) string {
    b := make([]byte, n)
    for i := range b {
        b[i] = letterBytes[rand.Intn(len(letterBytes))]
    }
    return string(b)
}

func generate_create() string {
	result := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<create>\n"
	number_of_content := rand.Intn(5)
	for i := 0; i < number_of_content; i++ {
		account_or_symbol := rand.Intn(2)
		if account_or_symbol == 0 {
			result += "<account id=\"" + strconv.Itoa(accountId) + "\" balance=\""+ strconv.Itoa(rand.Intn(2000))+ "\"/>\n"
			accountId++;
		}else{
			rand_str := RandStringBytes(1)
			//fmt.Println(accountId-100000)
			aid := rand.Intn(accountId-100000)
			aid += 100001
			amount := rand.Intn(100000) + 100
			result += "<symbol sym=\""+rand_str+"\">\n"
			result += "<account id=\""+strconv.Itoa(aid)+"\">"+strconv.Itoa(amount)+"</account>\n"
			result += "</symbol>\n"
		}
	}
	result += "</create>"
	length := len(result)
	result = strconv.Itoa(length) + "\n" + result
	return result
}

func generate_transaction() string {
	aid := rand.Intn(accountId-100000)
	aid += 100001
	result := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<transaction id=\""+strconv.Itoa(aid)+"\">\n"
	number_of_content := rand.Intn(5)
	for i := 0; i < number_of_content; i++ {
		rand_str := RandStringBytes(1)
		amt := rand.Intn(20)+1
		pos := rand.Intn(2)
		if pos == 0 {
			amt *= -1
		}
		price := rand.Intn(100)+1
		result += "<order sym=\""+rand_str+"\" amount=\""+strconv.Itoa(amt)+"\" limit=\""+strconv.Itoa(price)+"\"/>\n"
	}
	result += "</transaction>"
	length := len(result)
	result = strconv.Itoa(length) + "\n" + result
	return result
}

func generate() string {
	create_or_transaction := rand.Intn(2)
	if create_or_transaction == 1 {
		return generate_create()
	} else {
		return generate_transaction()
	}
}

func thread(num int, wg *sync.WaitGroup) {
	for i := 0; i < num; i++ {
		tmp_str := generate()
		sendString(tmp_str)
	}
	wg.Done()
}

func main() {
	args := os.Args
    if args == nil || len(args) != 3{
        return
    }
    n_thread, err := strconv.Atoi(args[1])
    checkError(err)
    n_request_per_thread, err := strconv.Atoi(args[2])
    checkError(err)
    var wg sync.WaitGroup
    now := time.Now()

    for i := 0; i < n_thread; i++ {
        wg.Add(1)
        go thread(n_request_per_thread, &wg)
    }
    wg.Wait()
	then := time.Now()
	diff := then.Sub(now)
    fmt.Println(diff)
}
