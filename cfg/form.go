package cfg

// The code in this file is responsible for building the CFG. Our approach is
// simple recursion, that is, we assume our branching depth will be low.
//
// As may be seen below, we form basic blocks by appending statement nodes
// until we hit a branching node (if, for, while). At that point, each
// branching node is used to create further edges.
//
// When recursing, we pass around "branch parent", which tell what is the
// "parent basic block" and how to reach it. See the example below, where this
// parameter ensures that when "4;" is evaluated, there is always an
// unconditional branch to "5;".
//
//     int a() {
//         1;
//	       if (true) {
//	           2;
//		       if (false) {
//			       3;
//			   }
//			   4;        <-- `pb' points towards the basic-block "5; return 0;"
//	       }
//	       5;
//	       return 0;
//     }
//
// Why do we need to define the branching kind in "branch parent"? With "if",
// we do not, however if we have a "while" statement, then the returning edge
// from a while-body is of the kind "while-false" -- we only depart from the
// while-body once the while-condition becomes false.
//
// Why do we need to pass around the node in "branch parent"? Later on, when we
// try to understand the generated CFG, we need to know what drives the
// branching. In practice, this means code generation.
//
// Note: Similar rationale applies to "branch loop" logic. We pass around a few
// closures, which generate suitable edges if "break" or "continue" are
// encountered.

import (
	"github.com/susji/c0/node"
)

type branchParent struct {
	to   *BasicBlock
	node node.Node
	how  BranchKind
}

type branchLoop struct {
	onBreak, onContinue func(*BasicBlock)
}

var blockEntry = BasicBlock{
	Id:         BLOCKID_ENTRY,
	Stmts:      Stmts{},
	Successors: []*Branch{},
}

var blockExit = &BasicBlock{
	Id:         BLOCKID_EXIT,
	Stmts:      Stmts{},
	Successors: []*Branch{},
}

func (bb *BasicBlock) newstmt(n node.Node) {
	bb.Stmts = append(bb.Stmts, n)
}

func (bb *BasicBlock) newsucc(rp *branchParent) {
	if rp.to == nil {
		rp.to = blockExit
	}
	branchid++
	bb.Successors = append(bb.Successors, &Branch{
		Id:   branchid,
		Kind: Kind{Node: rp.node, Kind: rp.how},
		From: bb,
		To:   rp.to,
	})
}

func (this *BasicBlock) newloop(n node.Node, body []node.Node,
	kt, kf BranchKind, rp *branchParent, left []node.Node, step node.Node) {
	afterloop := newblock()
	form(afterloop, rp, nil, left)
	// lb is the loop body itself.
	lb := newblock()
	// sb marks the end of loop body, which is always between the loop body and
	// the next iteration or breakoff. This is the place where the for loop's
	// step statement belongs.
	sb := newblock()
	ss := []node.Node{}
	if step != nil {
		ss = append(ss, step)
	}
	// As said above, the step body has a true-edge back to the loop body.
	form(sb, &branchParent{lb, n, kt}, nil, ss)
	// If we find a break or continue within the present loop, it means an
	// immediate (BK_ALWAYS) edge to post-loop or loop-start, respectively.
	lp := &branchLoop{
		onBreak: func(bb *BasicBlock) {
			bb.newsucc(&branchParent{afterloop, n, BK_ALWAYS})
		},
		onContinue: func(bb *BasicBlock) {
			bb.newsucc(&branchParent{sb, n, BK_ALWAYS})
		},
	}
	// As also said above, the loop body unconditionally connects to the step
	// body, which is always evaluated on each iteration.
	form(lb, &branchParent{sb, n, BK_ALWAYS}, lp, body)
	// Conditional false-edge after the step body.
	sb.newsucc(&branchParent{afterloop, n, kf})
	// Conditional true-edge to the loop body from the present block. This edge
	// means "enter the loop".
	this.newsucc(&branchParent{lb, n, kt})
	// Conditional false-edge from the present block over the loop. This edge
	// means "the loop is done".
	this.newsucc(&branchParent{afterloop, n, kf})
}

func extractbody(n node.Node) []node.Node {
	switch t := n.(type) {
	case *node.Block:
		return t.Value
	case nil:
		panic("nil body")
	default:
		return []node.Node{t}
	}
}

func (this *BasicBlock) newwhile(n *node.While, rp *branchParent, left []node.Node) {
	this.newloop(n, extractbody(n.Body), BK_WHILETRUE, BK_WHILEFALSE, rp, left, nil)
}

func (this *BasicBlock) newfor(n *node.For, rp *branchParent, left []node.Node) {
	this.newloop(n, extractbody(n.Body), BK_FORTRUE, BK_FORFALSE, rp, left, n.OnEach)
}

func (this *BasicBlock) newif(n *node.If, rp *branchParent, lp *branchLoop, left []node.Node) {
	// Continue evaluating the next basic block after this `if' branch. This
	// block then has to be found with edges after our True and False blocks.
	afterif := newblock()
	form(afterif, &branchParent{rp.to, n, BK_ALWAYS}, lp, left)

	// Recurse into the true-block.
	t := newblock()
	form(t, &branchParent{afterif, n, BK_ALWAYS}, lp, extractbody(n.True))
	this.newsucc(&branchParent{t, n, BK_IFTRUE})

	// Recurse into the false-block.
	if n.False != nil {
		f := newblock()
		form(f, &branchParent{afterif, n, BK_ALWAYS}, lp, extractbody(n.False))
		this.newsucc(&branchParent{f, n, BK_IFFALSE})
	} else {
		// If we are missing the `else' branch completely, then we have to
		// add an unconditional branch from this block.
		this.newsucc(&branchParent{afterif, n, BK_IFNOELSE})
	}
}

func form(b *BasicBlock, rp *branchParent, lp *branchLoop, left []node.Node) {
	for i, n := range left {
		switch t := n.(type) {
		case *node.If:
			b.newif(t, rp, lp, left[i+1:])
			return
		case *node.For:
			// XXX Form new basic block for initializer?
			b.newstmt(t.Init)
			b.newfor(t, rp, left[i+1:])
			return
		case *node.While:
			b.newwhile(t, rp, left[i+1:])
			return
		case *node.Return:
			b.newstmt(n)
			b.newsucc(&branchParent{blockExit, n, BK_ALWAYS})
			return
		case *node.Break:
			if lp == nil {
				panic("missing loop params on break")
			}
			lp.onBreak(b)
			b.newstmt(n)
			return
		case *node.Continue:
			if lp == nil {
				panic("missing loop params on continue")
			}
			lp.onContinue(b)
			b.newstmt(n)
			return
		default:
			b.newstmt(n)
		}
	}
	b.newsucc(rp)
}

func Form(fd *node.FunDef) (*CFG, []error) {
	c := &CFG{
		first:  blockEntry,
		fundef: fd,
	}
	second := newblock()
	c.first.newsucc(&branchParent{second, nil, BK_ALWAYS})
	// The initial parent basic block is the universal `blockExit'.
	form(second, &branchParent{blockExit, nil, BK_ALWAYS}, nil, fd.Body.Value)
	return c, nil
}
