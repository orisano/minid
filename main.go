package main

import (
	"fmt"
	"log"
	"os"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func compress(nodes []*parser.Node) [][]*parser.Node {
	var compressed [][]*parser.Node
	for _, node := range nodes {
		n := len(compressed)-1
		if n >= 0 && compressed[n][0].Value == node.Value {
			compressed[n] = append(compressed[n], node)
		} else {
			compressed = append(compressed, []*parser.Node{node})
		}
	}
	return compressed
}

func main() {
	result, err := parser.Parse(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	for _, nodes := range compress(result.AST.Children) {
		switch nodes[0].Value {
		case "run":
			fmt.Print(nodes[0].Original)
			for _, node := range nodes[1:] {
				fmt.Print(" && ", node.Next.Value)
			}
			fmt.Println()
		case "env":
			fmt.Print("ENV ")
			for _, node := range nodes {
				for n := node.Next; n != nil; n = n.Next.Next {
					key := n.Value
					val := n.Next.Value
					fmt.Print(key, "=", val, " ")
				}
			}
			fmt.Println()
		default:
			for _, node := range nodes {
				fmt.Println(node.Original)
			}
		}
	}
}
