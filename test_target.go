package main

import (
	"fmt"
)

func main() {
	s := "hoge"
	fmt.Println(fmt.Errorf("%v", s)) // これがSafeStringでラップされる
	fmt.Println(fmt.Errorf("%s", s)) // これもSafeStringでラップされる
	
	// Additional test cases
	msg := "error message"
	count := 42
	
	// These should be transformed
	fmt.Printf("Error: %v\n", fmt.Errorf("%v", msg))
	fmt.Printf("Error: %s\n", fmt.Errorf("%s", msg))
	
	// This should NOT be transformed (not a string)
	fmt.Printf("Count error: %v\n", fmt.Errorf("%d", count))
	
	// This should NOT be transformed (different format specifier)
	fmt.Printf("Different format: %v\n", fmt.Errorf("%d", count))
}