//go:build !windows && !darwin

package main

import "fmt"

func main() {
	fmt.Println("Degu Desktop is currently implemented for Windows.")
}
