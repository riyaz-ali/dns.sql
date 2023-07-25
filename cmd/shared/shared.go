package main

import (
	"go.riyazali.net/dns.sql"
	"go.riyazali.net/sqlite"
)

func init() { sqlite.Register(dns.ExtensionFunc()) }
func main() { /* nothing here fellas */ }
