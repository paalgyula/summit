package main

import "github.com/paalgyula/summit/pkg/summit/auth"

func main() {
	auth.StartServer("localhost:5000")
}
