package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

func main() {
	filePath := "foo"
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	for {
		buffer := make([]byte, 8)
		n, err := file.Read(buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Printf("error: %s\n", err.Error())
		}
		s := string(buffer[:n])
		fmt.Printf("read: %s\n", s)
	}
}
