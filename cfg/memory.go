package cfg

// These are used to memoize graph traversal to break loops.

type memblock map[BlockId]struct{}
type membranch map[BranchId]struct{}

func (mb memblock) add(bb *BasicBlock) {
	if mb.seen(bb) {
		return
	}
	mb[bb.Id] = struct{}{}
}

func (mb memblock) seen(bb *BasicBlock) bool {
	_, ok := mb[bb.Id]
	return ok
}

func (mb membranch) add(br *Branch) {
	if mb.seen(br) {
		return
	}
	mb[br.Id] = struct{}{}
}

func (mb membranch) seen(br *Branch) bool {
	_, ok := mb[br.Id]
	return ok
}
