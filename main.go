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
	"os/exec"
	"strings"
	"unicode"

	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("minid: ")
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

type flags struct {
	dockerfilePath string
	outputPath     string
}

type envEntry struct {
	key, value string
}

func run() error {
	var flags flags
	flag.StringVar(&flags.dockerfilePath, "f", "Dockerfile", "Dockerfile's path")
	flag.StringVar(&flags.outputPath, "o", "-", "generated Dockerfile path")
	flag.Parse()

	dockerfile, err := readDockerfile(flags.dockerfilePath)
	if err != nil {
		return fmt.Errorf("read Dockerfile: %w", err)
	}

	args := flag.Args()

	if len(args) >= 1 && args[0] == "build" {
		var buf bytes.Buffer
		if err := writeMinifiedDockerfile(&buf, dockerfile); err != nil {
			return fmt.Errorf("minify: %w", err)
		}
		cmd := exec.Command("docker", append([]string{"build", "-f", "-"}, args[1:]...)...)
		cmd.Stdin = &buf
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("exec docker build: %w", err)
		}
	} else {
		wc, err := openWriter(flags.outputPath)
		if err != nil {
			return fmt.Errorf("open Dockerfile writer: %w", err)
		}
		defer wc.Close()
		if err := writeMinifiedDockerfile(wc, dockerfile); err != nil {
			return fmt.Errorf("minify: %w", err)
		}
	}

	return nil
}

func openReader(name string) (io.ReadCloser, error) {
	if name == "-" {
		return os.Stdin, nil
	}
	return os.Open(name)
}

func openWriter(name string) (io.WriteCloser, error) {
	if name == "-" {
		return os.Stdout, nil
	}
	return os.Create(name)
}

func readDockerfile(dockerfilePath string) ([]byte, error) {
	rc, err := openReader(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("open reader: %w", err)
	}
	defer rc.Close()
	return ioutil.ReadAll(rc)
}

func writeMinifiedDockerfile(w io.Writer, dockerfile []byte) error {
	head, _, err := bufio.NewReader(bytes.NewReader(dockerfile)).ReadLine()
	if err != nil {
		return fmt.Errorf("head Dockerfile: %w", err)
	}

	result, err := parser.Parse(bytes.NewReader(dockerfile))
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	bw := bufio.NewWriter(w)
	if bytes.HasPrefix(head, []byte("# syntax =")) {
		fmt.Fprintln(bw, string(head))
	}
	for _, nodes := range compressBy(result.AST.Children, isSameCommand) {
		switch cmd := strings.ToUpper(nodes[0].Value); cmd {
		case "RUN":
			fmt.Fprint(bw, nodes[0].Original)
			for _, node := range nodes[1:] {
				fmt.Fprint(bw, " && ", node.Next.Value)
			}
			fmt.Fprintln(bw)
		case "ENV":
			var entries []envEntry
			for _, node := range nodes {
				for n := node.Next; n != nil; n = n.Next.Next {
					key := n.Value
					val := n.Next.Value
					entries = append(entries, envEntry{key: key, value: val})
				}
			}

			for len(entries) > 0 {
				fmt.Fprint(bw, cmd)
				vars := map[string]bool{}
				for len(entries) > 0 {
					e := entries[0]
					if hasReference(vars, e.value) {
						break
					}
					vars[e.key] = true
					fmt.Fprint(bw, " ", e.key, "=", e.value)
					entries = entries[1:]
				}
				fmt.Fprintln(bw)
			}
		case "LABEL":
			fmt.Fprint(bw, cmd)
			for _, node := range nodes {
				for n := node.Next; n != nil; n = n.Next.Next {
					key := n.Value
					val := n.Next.Value
					fmt.Fprint(bw, " ", key, "=", val)
				}
			}
			fmt.Fprintln(bw)
		case "ADD", "COPY":
			for _, xs := range compressBy(nodes, isSameDestination) {
				fmt.Fprint(bw, cmd)
				if len(xs[0].Flags) > 0 {
					fmt.Fprint(bw, " ", strings.Join(xs[0].Flags, " "))
				}
				for _, x := range xs {
					for n := x.Next; n.Next != nil; n = n.Next {
						fmt.Fprint(bw, " ", n.Value)
					}
				}
				fmt.Fprint(bw, " ", destination(xs[0]))
				fmt.Fprintln(bw)
			}
		default:
			for _, node := range nodes {
				fmt.Fprintln(bw, node.Original)
			}
		}
	}
	if err := bw.Flush(); err != nil {
		return err
	}
	return nil
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
	switch strings.ToLower(n.Value) {
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
