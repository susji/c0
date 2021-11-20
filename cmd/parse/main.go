// parse is a simple command-line-based parser for C0. It is mainly intended
// for quick and dirty testing.
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/susji/c0/analyze"
	"github.com/susji/c0/cfg"
	"github.com/susji/c0/lex"
	"github.com/susji/c0/node"
	"github.com/susji/c0/parse"
)

func fatal(f string, va ...interface{}) {
	fmt.Fprintf(os.Stderr, "fatal: "+f+"\n", va...)
	os.Exit(1)
}

func perr(f string, va ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: "+f+"\n", va...)
}

func note(f string, va ...interface{}) {
	fmt.Fprintf(os.Stdout, "[] "+f+"\n", va...)
}

func dumper(n node.Node, depth int) bool {
	i := ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
	ie := 4 * depth
	if ie > len(i)-1 {
		ie = len(i) - 1
	}
	fmt.Printf("%s %s\n", i[0:ie], n)
	return true
}

func tap(dumptoks bool, src []rune, p *parse.Parser, dumpcfg bool) {
	toks, errs := lex.Lex(src)
	if errs != nil {
		perr("lexing: %s\n", errs)
		return
	}
	if dumptoks {
		fmt.Println(toks)
	}
	for toks.Len() > 0 {
		err := p.Parse(toks)
		if err != nil {
			for _, e := range p.Errors() {
				perr("parse: %s", e)
			}
		}
		nodes := p.Nodes()
		note("%d nodes", len(nodes))
		for ni, n := range nodes {
			fmt.Printf("{%d}\n", ni)
			node.Walk(n, dumper)
		}
		note("syntax errors")
		a := analyze.New(p.Fn())
		aerrs := a.Analyze(p.Nodes())
		for _, aerr := range aerrs {
			perr("analyze: %s", aerr)
		}
		for _, n := range p.Nodes() {
			switch t := n.(type) {
			case *node.FunDef:
				note("CFG for function %q", t.FunDecl.Name)
				cfg, cerrs := cfg.Form(t)
				if len(cerrs) > 0 {
					for _, cerr := range cerrs {
						perr("cfg: %s", cerr)
					}
					break
				}
				tf, err := ioutil.TempFile("", "ccdot*")
				if err != nil {
					panic(err)
				}
				// XXX Ignoring errors
				tf.WriteString(cfg.Dot())
				note("wrote dot: %s", tf.Name())
				tf.Close()
			}
		}
	}
}

func doloop(dumptoks bool) {
	r := bufio.NewReader(os.Stdin)
	i := 0
	for {
		fmt.Printf("[%d] >> ", i)
		line, err := r.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Bailing...\n")
			os.Exit(0)
		}
		tap(dumptoks, []rune(strings.TrimSpace(line)), parse.New(), false)
		i++
	}
}

func main() {
	dumptoks := flag.Bool("dumptoks", false, "dump lexed tokens")
	dofile := flag.String("file", "", "parse and dump a .c0 file")
	dumpcfg := flag.Bool("dumpcfg", false, "dump CFG as dot (stderr)")
	flag.Parse()

	if *dofile != "" {
		src, err := ioutil.ReadFile(*dofile)
		if err != nil {
			fatal("cannot open %s: %s\n", *dofile, err)
		}
		tap(*dumptoks, bytes.Runes(src), parse.NewFile(*dofile), *dumpcfg)
	} else {
		if *dumpcfg {
			fatal("cannot dump dot with repl")
		}
		doloop(*dumptoks)
	}
}
