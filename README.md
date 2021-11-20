# c0

This project is a small compiler based on the specifications of the C0
Reference documentation. Regardless of the name, it attempts to implement the
C1 extensions.

## Design

The compiler is currently architected into a traditional "lex, parse, analyze"
form. Next stages are to generate a CFG from the verified forest of tree nodes,
perform SSA, and implement code generation.

### Lexing

Less traditionally, the lexing stage is implemented with parser combinators.
The original plan was to implement the whole lexing & parsing in this manner,
however the author did not manage to implement this in a satisfactory manner.
Remnants of that era may still be seen in the API of `primitives`, which should
be pruned to match the simpler needs of plain lexing.

### Parsing

Parsing attempts to match the grammar of the C0 Reference, however some
liberties might have been taken when we have modified it to ease parsing. See
the parsing of type declarations for an example. Otherwise the parsing approach
is straightforward recursive-descent with precedence climbing for expressions.
A noteworthy detail is "Node tagging", which means that each parsed `Node` will
receive an unique identifier, which is heavily used in syntax & type-checking.

### Analysis

Syntax and type checking are performed with a single pass over the parsed
nodes. During parsing, we tagged each `Node` with an unique identifier, and
these IDs act as our primary keys for all syntax & type checking operations.
See the contents of the `Syntax` struct and the implementation of its `check*`
functions for details.

### Control-flow graph

The control-flow graph is formed with a simple recursive algorithm. We do not
aim for or reach a maximal basic block representation.

### SSA form and the intermediate language

We use a simplistic approach for the intermediate language, that is, we mostly
ape from LLVM and stick with basics. We do not aim for minimal phi generation.

## Contracts

The C0 Reference specifies contracts, which make use of special comment blocks.
Presently we ignore them, but we may implement the runtime support once we are
at a stage where things work otherwise. Noteworthily, our parsing approach
already accomodates them -- see `Peek` vs. `PeekAll`.

## Tests

See `*/*_test.go` and `go test ./...`. Please note that the tests for the
`parse` package intentionally turn off the `Node` identifier-generation logic.
See `parse/parse_test.go:TestMain`.

## TODO

[] Lexing: skip on errors and try munching something else

[] Errors: Generalize error-gathering from different stages and report them
   cohesively

[] Syntax: Exactly one `int main()` function is always needed

[] Parsing: warn about lone identifiers which may be undefined typedefs
   - Stmt vs. SimpleStmt vs. Expr

[] SSA: Figure out how our RISC-like IR should look like

[] Codegen: Target x86 and/or x86-64 with System V ABI on Linux (we also need to handle C compatibility to *some* degree)

[] Optimizations: Figure out which optimizations at which stage
