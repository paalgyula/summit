package main

import "github.com/paalgyula/summit/pkg/blizzard/auth"

func main() {
	auth.StartServer("localhost:5000")
}
