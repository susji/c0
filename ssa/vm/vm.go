package vm

import (
	"fmt"
	"strings"

	"github.com/susji/c0/ir"
	"github.com/susji/c0/ssa"
)

type VM struct {
	funcs map[string]*ssa.SSA
	regs  map[ir.Variable]int32
	mem   []int32
}

func New() *VM {
	return &VM{
		funcs: map[string]*ssa.SSA{},
		regs:  map[ir.Variable]int32{},
		mem:   []int32{},
	}
}

func (vm *VM) Insert(name string, s *ssa.SSA) {
	vm.funcs[name] = s
}

func (vm *VM) Inst(name, f string, va ...interface{}) {
	fmt.Printf(fmt.Sprintf("%-10s | ", name)+f+"\n", va...)
}

func (vm *VM) Load(from *ir.Variable, to *ir.Variable) {
	fi := vm.regs[*from]
	vm.regs[*to] = vm.mem[fi]
}

func (vm *VM) Set(to *ir.Variable, what ir.Value) {
	var val int32
	switch t := what.(type) {
	case *ir.Numeric32i:
		val = t.Value
	case *ir.Variable:
		val = vm.regs[*t]
	}
	vm.regs[*to] = val
}

func (vm *VM) Store(variable, value *ir.Variable) {
	ptr := vm.regs[*variable]
	vm.mem[ptr] = vm.regs[*value]
}

func (vm *VM) Alloca() int32 {
	vm.mem = append(vm.mem, 0)
	return int32(len(vm.mem) - 1)
}

func (vm *VM) ExtractValue(v ir.Value) int32 {
	fmt.Println("Extracting value:", v)
	switch t := v.(type) {
	case *ir.Variable:
		return vm.regs[*t]
	case *ir.Numeric32i:
		return t.Value
	default:
		panic("zzz")
	}
}

func (vm *VM) BinOp(to *ir.Variable, left, right ir.Value, op func(v1, v2 int32) int32) {
	l := vm.ExtractValue(left)
	r := vm.ExtractValue(right)
	vm.Set(to, &ir.Numeric32i{Value: op(l, r)})
}

func (vm *VM) DumpMem() string {
	b := &strings.Builder{}
	b.WriteString("# memory\n")
	for mem, val := range vm.mem {
		b.WriteString(fmt.Sprintf("%10d = %d\n", mem, val))
	}
	return b.String()
}

func (vm *VM) DumpRegs() string {
	b := &strings.Builder{}
	b.WriteString("# registers\n")
	for mem, val := range vm.regs {
		b.WriteString(fmt.Sprintf("%10s = %d\n", mem.String(), val))
	}
	return b.String()
}

func (vm *VM) Run(verbose bool) *int32 {
	ret := new(int32)
	for fun, fus := range vm.funcs {
		fmt.Println("# func:", fun)
		for _, inst := range fus.Instructions {
			switch t := inst.(type) {
			case ir.Alloca:
				vm.Inst("alloca", "%s", t.To)
				vm.regs[*t.To] = vm.Alloca()
			case ir.Mov:
				vm.Inst("mov", "%s -> %s", t.What, t.To)
				vm.Set(t.To, t.What)
			case ir.Store:
				vm.Inst("store", "%s -> [%s]", t.From, t.To)
				vm.Store(t.To, t.From)
			case ir.Load:
				vm.Inst("load", "[%s] -> %s", t.From, t.To)
				vm.Load(t.From, t.To)
			case ir.Add:
				vm.Inst("add", "%s = %s + %s", t.To, t.Left, t.Right)
				vm.BinOp(t.To, t.Left, t.Right, func(v1, v2 int32) int32 {
					return v1 + v2
				})
			case ir.Mul:
				vm.Inst("mul", "%s = %s * %s", t.To, t.Left, t.Right)
				vm.BinOp(t.To, t.Left, t.Right, func(v1, v2 int32) int32 {
					return v1 * v2
				})
			case ir.Xor:
				vm.Inst("xor", "%s = %s ^ %s", t.To, t.Left, t.Right)
				vm.BinOp(t.To, t.Left, t.Right, func(v1, v2 int32) int32 {
					return v1 ^ v2
				})
			case ir.Return:
				vm.Inst("return", "%s", t.With)
				*ret = vm.ExtractValue(t.With)
				break
			case ir.Label:
				vm.Inst("label", "%s", t.Name)
			default:
				panic(fmt.Sprintf("unknown instruction: %s", inst))
			}
			if verbose {
				fmt.Println(vm.DumpMem())
				fmt.Println(vm.DumpRegs())
			}
		}
	}
	return ret
}
