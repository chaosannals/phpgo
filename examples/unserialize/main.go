package main

import (
	"fmt"
	"log"
	"os"

	"github.com/chaosannals/phpgo"
)

type ObjectDemoE struct {
	C string  `php:"c"`
	D []int   `php:"\x00Demo\\Ns\\DemoE\x00d"` // 私有变量特有命名规则
	F float64 `php:"f"`
	N any     `php:"n"`
}

type ObjectDemoA struct {
	A    string         `php:"\x00Demo\\Ns\\DemoA\x00a"` // 私有变量特有命名规则
	B    int            `php:"b"`
	List []int          `php:"list"`
	M    map[string]any `php:"m"`
	E    ObjectDemoE    `php:"\x00Demo\\Ns\\DemoA\x00e"` // 私有变量特有命名规则
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

	// 强类型解析
	var demo ObjectDemoA
	if err := phpgo.UnSerializeTo(content, &demo); err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("demo: %v\n", demo)
}
