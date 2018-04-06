package main

import (
  "fmt"
  "os"
  "encoding/xml"
  "strconv"
  "bufio"
  "net"
)


// Create

type Account struct {
  XMLName xml.Name `xml:"account"`
  Id      string   `xml:"id,attr"`
  Balance string   `xml:"balance,attr"`
}

type Symbol struct {
  XMLName  xml.Name `xml:"symbol"`
  Sym      string   `xml:"sym,attr"`
  Accounts []struct {
    Id     string `xml:"id,attr"`
    Amount string `xml:",innerxml"`
    } `xml:"account"`
  }

  // Transactions

func dial(req string) (status string, err error){
  conn, err := net.Dial("tcp", "localhost:12345")
  if err != nil {
	return
  }
  fmt.Fprintf(conn, req)
  status, err = bufio.NewReader(conn).ReadString('\n')
  return
}
  func main() {
program:
    for {
      var req = xml.Header

      fmt.Fprintf(os.Stdout,"Enter one of the following:\n\n-create\n-transactions\n-exit\n\n")
      var cmd string
      fmt.Scanf("%s", &cmd)


      switch cmd {
      case "create":
        req += "<create>\n"

outer:
        for {
          fmt.Fprintf(os.Stdout,"Enter one of the following:\n\n-account\n-symbol\n-done\n\n")
          var c_cmd string
          fmt.Scanf("%s", &c_cmd)

          switch c_cmd {
          case "account":
            fmt.Fprintf(os.Stdout,"Account ID:\n")
            var acct_id string
            fmt.Scanf("%s", &acct_id)

            fmt.Fprintf(os.Stdout,"Balance:\n")
            var bal string
            fmt.Scanf("%s", &bal)

            acct := Account{Id: acct_id, Balance: bal}
            if acct_string, err := xml.MarshalIndent(acct, "", "    "); err == nil {
              req += string(acct_string) + "\n"
            }
          case "symbol":
          case "done":
            break outer
          default:

          }

        }
        len_req := strconv.Itoa((len(req)+1))
        req = len_req + "\n" + req
        req += "</create>"
        fmt.Fprintf(os.Stdout, "%s\n", req)
        status, err := dial(req)
        fmt.Fprintf(os.Stdout, "status: %s\nerror: %s\n", status, err)

      case "transactions":

      case "exit":
        break program

      default:
      }
    }
  }
