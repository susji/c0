package cfg

// The contents of this file are responsible for finding connections between
// nodes in a CFG. "A connection" is now meant to mean a directed path with a
// start and end. As these starts and ends are sought with caller-provided
// callbacks, there should be enough flexibility.

import (
	"github.com/susji/c0/node"
)

type NodeCb func(n node.Node) bool

func nodeinblock(cb NodeCb, i int, b *BasicBlock) int {
	for j := i; j < len(b.Stmts); j++ {
		if cb(b.Stmts[j]) {
			return j + 1
		}
	}
	return -1
}

func nodesinblock(start, end NodeCb, b *BasicBlock) (int, int) {
	istart := nodeinblock(start, 0, b)
	var startend int
	if istart == -1 {
		startend = 0
	} else {
		startend = istart
	}
	iend := nodeinblock(end, startend, b)
	return istart, iend
}

func connect(start, end NodeCb, b *BasicBlock, mem membranch) bool {
	if start == nil {
		// start may be nil for two reasons:
		//
		//    1) We want to find the end from function start
		//    2) We already found it on previous call to connect
		//
		// In any case, now the problem is simpler as we do not have to worry
		// about the possibility of finding start and end from the same basic
		// block. That case is handled below in the else branch.
		//
		if nodeinblock(end, 0, b) > -1 {
			return true
		}
		for _, succ := range b.Successors {
			if mem.seen(succ) {
				continue
			}
			mem.add(succ)
			if connect(nil, end, succ.To, mem) {
				return true
			}
		}
		return false
	} else {
		istart, iend := nodesinblock(start, end, b)
		switch {
		case istart > -1 && iend > -1:
			// Both nodes discovered in the present same basic block -> done.
			return true
		case istart > -1:
			// Only the start node found in the present basic block -> recurse
			// in attempt to find end -- note the start nil-ness.
			for _, succ := range b.Successors {
				if mem.seen(succ) {
					continue
				}
				mem.add(succ)
				if connect(nil, end, succ.To, mem) {
					return true
				}
			}
		case iend > -1:
			// If we find the end node and start is nowhere to be seen, we know
			// that forming the connection failed.
			return false
		default:
			// Neither found -> recurse harder.
			for _, succ := range b.Successors {
				if mem.seen(succ) {
					continue
				}
				mem.add(succ)
				if connect(start, end, succ.To, mem) {
					return true
				}
			}
		}
		return false
	}
}

// Connect is used to determine whether there is at least one possible
// branching path for two nodes. If start is nil, it is interpreted as function
// start.
func (c *CFG) Connect(start, end NodeCb) bool {
	if end == nil {
		panic("no end cb")
	}
	return connect(start, end, &c.first, membranch{})
}
