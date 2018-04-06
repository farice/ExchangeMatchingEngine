package main

import (
  "encoding/xml"
)
// Remember to capitalize field names so they are exported

type Account struct {
	XMLName xml.Name `xml:"account"`
	Id      string   `xml:"id,attr"`
	Balance string   `xml:"balance,attr"`
}

type Dump struct {
  XMLName xml.Name `xml:"dump"`
}

type Symbol struct {
	XMLName  xml.Name `xml:"symbol"`
	Sym      string   `xml:"sym,attr"`
	Accounts []struct {
		Id     string `xml:"id,attr"`
		Amount string `xml:",innerxml"`
	} `xml:"account"`
}

type Order struct {
	XMLName xml.Name `xml:"order"`
	Sym     string   `xml:"sym,attr"`
	Amount  string   `xml:"amount,attr"` // negative means to sell
	Limit   string   `xml:"limit,attr"`
}

type Cancel struct {
	XMLName       xml.Name `xml:"cancel"`
	TransactionID string   `xml:"id,attr"`
}

type Query struct {
	XMLName       xml.Name `xml:"query"`
	TransactionID string   `xml:"id,attr"`
}

type OpenQueryResponse struct {
	XMLName xml.Name `xml:"open"`
	Shares string   `xml:"shares,attr"`
}

type CancelQueryResponse struct {
	XMLName xml.Name `xml:"canceled"`
	Shares string   `xml:"shares,attr"`
	Time string `xml:"time,attr"`
}

type ExecutedQueryResponse struct {
	XMLName xml.Name `xml:"executed"`
	Shares string   `xml:"shares,attr"`
	Price string   `xml:"price,attr"`
	Time string `xml:"time,attr"`
}

type OpenResponse struct {
	XMLName       xml.Name `xml:"opened"`
	TransactionID string   `xml:"id,attr"`
	Sym           string   `xml:"sym,attr"`
	Amount        string   `xml:"amount,attr"` // negative means to sell
	Limit         string   `xml:"limit,attr"`
}

type ErrorTransResponse struct {
	XMLName xml.Name `xml:"error"`
	Sym     string   `xml:"sym,attr"`
	Amount  string   `xml:"amount,attr"` // negative means to sell
	Limit   string   `xml:"limit,attr"`
	Reason  string   `xml:",innerxml"`
}

type CreatedResponse struct {
	XMLName xml.Name `xml:"created"`
	Sym     string   `xml:"sym,attr,omitempty"`
	Id      string   `xml:"id,attr,omitempty"`
}

type ErrorCreateResponse struct {
	XMLName xml.Name `xml:"error"`
	Sym     string   `xml:"sym,attr,omitempty"`
	Id      string   `xml:"id,attr,omitempty"`
	Reason  string   `xml:",innerxml"`
}

type ErrorQueryCancelResponse struct {
  XMLName xml.Name `xml:"error"`
  TransactionID string   `xml:"id,attr"`
  Reason  string   `xml:",innerxml"`
}
