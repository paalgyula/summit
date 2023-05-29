package main

import "github.com/paalgyula/summit/pkg/summit/auth"

func main() {
	auth.NewServer("localhost:5000")
}
