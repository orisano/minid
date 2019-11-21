package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"unicode"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

type entry struct {
	key, value string
}

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
	outputPath := flag.String("o", "-", "generated Dockerfile path")
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
	var w io.Writer
	if *outputPath == "-" {
		w = os.Stdout
	} else {
		f, err := os.Create(*outputPath)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		w = f
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}

	head, _, err := bufio.NewReader(bytes.NewReader(b)).ReadLine()
	if err != nil {
		log.Fatal(err)
	}
	result, err := parser.Parse(bytes.NewReader(b))
	if err != nil {
		log.Fatal(err)
	}
	if bytes.HasPrefix(head, []byte("# syntax =")) {
		fmt.Fprintln(w, string(head))
	}
	for _, nodes := range compressBy(result.AST.Children, isSameCommand) {
		switch cmd := strings.ToUpper(nodes[0].Value); cmd {
		case "RUN":
			fmt.Fprint(w, nodes[0].Original)
			for _, node := range nodes[1:] {
				fmt.Fprint(w, " && ", node.Next.Value)
			}
			fmt.Fprintln(w)
		case "ENV":
			var entries []entry
			for _, node := range nodes {
				for n := node.Next; n != nil; n = n.Next.Next {
					key := n.Value
					val := n.Next.Value
					entries = append(entries, entry{key: key, value: val})
				}
			}

			for len(entries) > 0 {
				fmt.Fprint(w, cmd)
				vars := map[string]bool{}
				for len(entries) > 0 {
					e := entries[0]
					if hasReference(vars, e.value) {
						break
					}
					vars[e.key] = true
					fmt.Fprint(w, " ", e.key, "=", e.value)
					entries = entries[1:]
				}
				fmt.Fprintln(w)
			}
		case "LABEL":
			fmt.Fprint(w, cmd)
			for _, node := range nodes {
				for n := node.Next; n != nil; n = n.Next.Next {
					key := n.Value
					val := n.Next.Value
					fmt.Fprint(w, " ", key, "=", val)
				}
			}
			fmt.Fprintln(w)
		case "ADD", "COPY":
			for _, xs := range compressBy(nodes, isSameDestination) {
				fmt.Fprint(w, cmd)
				if len(xs[0].Flags) > 0 {
					fmt.Fprint(w, " ", strings.Join(xs[0].Flags, " "))
				}
				for _, x := range xs {
					for n := x.Next; n.Next != nil; n = n.Next {
						fmt.Fprint(w, " ", n.Value)
					}
				}
				fmt.Fprintln(w, " ", destination(xs[0]))
			}
		default:
			for _, node := range nodes {
				fmt.Fprintln(w, node.Original)
			}
		}
	}
}

func hasReference(vars map[string]bool, expr string) bool {
	if !strings.ContainsRune(expr, '$') {
		return false
	}
	for v := range vars {
		if strings.Contains(expr, "${"+v+"}") {
			return true
		}
		if strings.Contains(expr, "${"+v+":") {
			return true
		}
		tokens := strings.Split(expr, "$"+v)
		if len(tokens) == 1 {
			continue
		}
		for i := 1; i < len(tokens); i++ {
			rs := []rune(tokens[i])
			if len(rs) == 0 {
				return true
			}
			r := rs[0]
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
				return true
			}
		}
	}
	return false
}
