package main

import (
	"fmt"
	"log"
	"os"

	"github.com/chaosannals/phpgo"
)

type ObjectDemo struct {
	A string `php:"a"`
	B int    `php:"b"`
}

func main() {
	content, err := os.ReadFile("demo.txt")
	if err != nil {
		log.Fatalln(err)
	}

	// 弱类型解析
	if demo, err := phpgo.UnSerialize(content); err != nil {
		log.Fatalln(err)
	} else {
		fmt.Printf("demo: %v\n", demo)
	}

	// TODO 强类型解析
	// var demo ObjectDemo
}
