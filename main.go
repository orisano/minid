package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func compressBy(nodes []*parser.Node, pred func(a, b *parser.Node) bool) [][]*parser.Node {
	var compressed [][]*parser.Node
	for _, node := range nodes {
		n := len(compressed) - 1
		if n >= 0 && pred(compressed[n][0], node) {
			compressed[n] = append(compressed[n], node)
		} else {
			compressed = append(compressed, []*parser.Node{node})
		}
	}
	return compressed
}

func isSameCommand(a, b *parser.Node) bool {
	return a.Value == b.Value
}

func destination(n *parser.Node) string {
	switch n.Value {
	case "add", "copy":
		x := n.Next
		for x.Next != nil {
			x = x.Next
		}
		return x.Value
	default:
		panic("unexpected node: " + n.Value)
	}
}

func isSameDestination(a, b *parser.Node) bool {
	if strings.Join(a.Flags, "") != strings.Join(b.Flags, "") {
		return false
	}
	return destination(a) == destination(b)
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
	for _, nodes := range compressBy(result.AST.Children, isSameCommand) {
		switch cmd := strings.ToUpper(nodes[0].Value); cmd {
		case "RUN":
			fmt.Print(nodes[0].Original)
			for _, node := range nodes[1:] {
				fmt.Print(" && ", node.Next.Value)
			}
			fmt.Println()
		case "ENV":
			fmt.Print(cmd)
			for _, node := range nodes {
				for n := node.Next; n != nil; n = n.Next.Next {
					key := n.Value
					val := n.Next.Value
					fmt.Print(" ", key, "=", val)
				}
			}
			fmt.Println()
		case "ADD", "COPY":
			for _, xs := range compressBy(nodes, isSameDestination) {
				fmt.Print(cmd)
				if len(xs[0].Flags) > 0 {
					fmt.Print(" ", strings.Join(xs[0].Flags, " "))
				}
				for _, x := range xs {
					for n := x.Next; n.Next != nil; n = n.Next {
						fmt.Print(" ", n.Value)
					}
				}
				fmt.Println(" ", destination(xs[0]))
			}
		default:
			for _, node := range nodes {
				fmt.Println(node.Original)
			}
		}
	}
}
