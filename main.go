package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func compress(nodes []*parser.Node) [][]*parser.Node {
	var compressed [][]*parser.Node
	for _, node := range nodes {
		n := len(compressed) - 1
		if n >= 0 && compressed[n][0].Value == node.Value {
			compressed[n] = append(compressed[n], node)
		} else {
			compressed = append(compressed, []*parser.Node{node})
		}
	}
	return compressed
}

func main() {
	var dockerfilePath string
	flag.StringVar(&dockerfilePath, "f", "Dockerfile", "Dockerfile's path")
	flag.Parse()

	var r io.Reader
	if dockerfilePath == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(dockerfilePath)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		r = f
	}

	result, err := parser.Parse(r)
	if err != nil {
		log.Fatal(err)
	}
	for _, nodes := range compress(result.AST.Children) {
		switch nodes[0].Value {
		case "run":
			fmt.Print(nodes[0].Original)
			for _, node := range nodes[1:] {
				fmt.Print("; ", node.Next.Value)
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
