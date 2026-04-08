package analyzers

import (
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

// bufferPeekStoreAnalyzer returns an analyzer that flags bytes.Buffer.Peek()
// results used after the buffer is mutated. Peek returns a slice aliasing
// internal buffer memory, so any mutation invalidates it. Using the slice
// after mutation is silent data corruption.
func bufferPeekStoreAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "bufferpeekstore",
		Doc:      "flags bytes.Buffer.Peek() results used after buffer mutation; the returned slice aliases internal memory and becomes invalid",
		Run:      runBufferPeekStore,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	}
}

// bufferMutators lists every *bytes.Buffer method that modifies internal
// memory, invalidating any slice returned by Peek.
var bufferMutators = map[string]bool{ //nolint:gochecknoglobals // package-internal lookup table, not mutable state
	"Write":       true,
	"WriteString": true,
	"WriteByte":   true,
	"WriteRune":   true,
	"ReadFrom":    true,
	"Reset":       true,
	"Truncate":    true,
	"Grow":        true,
	"Read":        true,
	"ReadByte":    true,
	"ReadRune":    true,
	"ReadBytes":   true,
	"ReadString":  true,
	"Next":        true,
}

func runBufferPeekStore(pass *analysis.Pass) (any, error) {
	ssaResult, castOK := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	for _, fn := range ssaResult.SrcFuncs {
		checkFunctionForPeekStore(pass, fn)
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkFunctionForPeekStore scans a single SSA function for Peek calls whose
// returned slice is used after the same buffer is mutated.
func checkFunctionForPeekStore(pass *analysis.Pass, fn *ssa.Function) {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			call, isCall := instr.(*ssa.Call)
			if !isCall {
				continue
			}

			callee := call.Call.StaticCallee()
			if callee == nil {
				continue
			}

			if !isBufferMethod(callee, "Peek") {
				continue
			}

			// Peek's receiver is the first argument in SSA form.
			if len(call.Call.Args) == 0 {
				continue
			}

			receiver := call.Call.Args[0]

			// Peek returns ([]byte, error), so the call produces a tuple.
			// The []byte result is accessed via Extract #0 instructions.
			sliceValues := extractPeekSliceValues(call)
			if len(sliceValues) == 0 {
				continue
			}

			if peekSliceUsedAfterMutation(call, receiver, sliceValues) {
				pass.Reportf(call.Pos(), "bytes.Buffer.Peek result used after buffer mutation; the returned slice aliases internal memory and is now invalid")
			}
		}
	}
}

// extractPeekSliceValues finds all SSA values representing the []byte
// component of a Peek call's return tuple. Since Peek returns ([]byte, error),
// the call produces a tuple and Extract #0 yields the slice.
func extractPeekSliceValues(call *ssa.Call) []ssa.Value {
	refs := call.Referrers()
	if refs == nil {
		return nil
	}

	var sliceVals []ssa.Value

	for _, ref := range *refs {
		ext, isExtract := ref.(*ssa.Extract)
		if !isExtract {
			continue
		}

		// Index 0 is the []byte result, index 1 is the error.
		if ext.Index == 0 {
			sliceVals = append(sliceVals, ext)
		}
	}

	return sliceVals
}

// peekSliceUsedAfterMutation checks whether any of the slice values derived
// from a Peek call are referenced after a mutation to the same buffer.
//
// Strategy: for each basic block containing a mutation of the same buffer
// receiver, check whether that mutation dominates any block where the slice
// value is used, or whether mutation and use are in the same block with the
// mutation appearing first.
func peekSliceUsedAfterMutation(peekCall *ssa.Call, receiver ssa.Value, sliceVals []ssa.Value) bool {
	fn := peekCall.Parent()

	// Collect every mutation instruction targeting the same buffer receiver.
	mutations := collectBufferMutations(fn, receiver)
	if len(mutations) == 0 {
		return false
	}

	// Collect every instruction that uses one of the Peek slice values.
	uses := collectSliceUses(sliceVals)
	if len(uses) == 0 {
		return false
	}

	peekBlock := peekCall.Block()

	for _, mut := range mutations {
		// Skip mutations that happen before the Peek in the same block.
		if mut.Block() == peekBlock {
			if instrIndex(mut) <= instrIndex(peekCall) {
				continue
			}
		}

		for _, use := range uses {
			if mutationPrecedesUse(mut, use) {
				return true
			}
		}
	}

	return false
}

// collectBufferMutations finds all *ssa.Call instructions in fn that call a
// mutating method on the same buffer receiver.
func collectBufferMutations(fn *ssa.Function, receiver ssa.Value) []ssa.Instruction {
	var mutations []ssa.Instruction

	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			call, isCall := instr.(*ssa.Call)
			if !isCall {
				continue
			}

			callee := call.Call.StaticCallee()
			if callee == nil {
				continue
			}

			if !bufferMutators[callee.Name()] {
				continue
			}

			if !isBufferMethod(callee, callee.Name()) {
				continue
			}

			if len(call.Call.Args) == 0 {
				continue
			}

			// Same buffer receiver check: compare underlying SSA values.
			if call.Call.Args[0] == receiver {
				mutations = append(mutations, instr)
			}
		}
	}

	return mutations
}

// collectSliceUses gathers every instruction that references one of the
// Peek-derived slice values, excluding Extract instructions themselves
// (they are part of the tuple decomposition, not actual usage).
func collectSliceUses(sliceVals []ssa.Value) []ssa.Instruction {
	var uses []ssa.Instruction

	for _, sv := range sliceVals {
		refs := sv.Referrers()
		if refs == nil {
			continue
		}

		for idx := range *refs {
			ref := (*refs)[idx]

			// Exclude Extracts and DebugRef since they aren't real usage.
			switch ref.(type) {
			case *ssa.Extract, *ssa.DebugRef:
				continue
			}

			uses = append(uses, ref)
		}
	}

	return uses
}

// mutationPrecedesUse reports whether a mutation instruction happens before
// a use instruction in execution order. Checks two cases:
//  1. Same block: mutation has a lower instruction index than the use.
//  2. Cross-block: the mutation's block dominates the use's block.
func mutationPrecedesUse(mutation, use ssa.Instruction) bool {
	mutBlock := mutation.Block()
	useBlock := use.Block()

	if mutBlock == useBlock {
		return instrIndex(mutation) < instrIndex(use)
	}

	return mutBlock.Dominates(useBlock)
}

// instrIndex returns the position of instr within its basic block's
// instruction list. Returns -1 if not found (should not happen for valid SSA).
func instrIndex(instr ssa.Instruction) int {
	block := instr.Block()
	if block == nil {
		return -1
	}

	for idx, candidate := range block.Instrs {
		if candidate == instr {
			return idx
		}
	}

	return -1
}

// isBufferMethod reports whether callee is a method named methodName declared
// on *bytes.Buffer.
func isBufferMethod(callee *ssa.Function, methodName string) bool {
	if callee.Name() != methodName {
		return false
	}

	sig := callee.Signature

	recv := sig.Recv()
	if recv == nil {
		return false
	}

	recvType := recv.Type()

	ptr, isPtr := recvType.(*types.Pointer)
	if !isPtr {
		return false
	}

	named, isNamed := ptr.Elem().(*types.Named)
	if !isNamed {
		return false
	}

	obj := named.Obj()

	return obj.Pkg() != nil && obj.Pkg().Path() == "bytes" && obj.Name() == "Buffer"
}
