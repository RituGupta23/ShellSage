// Every go file starts with package declaration
// package main - It tells Go "It is an executable program" not a library
// main function in the package main is the entry point (where our program starts running)

package main

// import is used to import packages
// fmt - Used to format and print output
// os - Used to access operating system functionality
import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello World")
	fmt.Println(os.Args)

	if (len(os.Args)< 2) {
		fmt.Fprintln(os.Stderr, "Usage: sg \"your query here\"")
		os.Exit(1)
	}

	query := os.Args[1]
	fmt.Printf("ShellSage: you asked: %s\n", query)
}