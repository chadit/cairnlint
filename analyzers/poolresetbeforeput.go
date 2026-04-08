package analyzers

import (
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

// syncPkgPath is the import path for the sync package, extracted as a constant
// to satisfy goconst across analyzers that check sync types.
const syncPkgPath = "sync"

// poolPutMinArgs is the minimum number of arguments for a (*sync.Pool).Put
// call in SSA form: arg[0] is the pool receiver, arg[1] is the object.
const poolPutMinArgs = 2

// poolResetBeforePutAnalyzer returns an analyzer that flags sync.Pool.Put
// calls where the object hasn't been reset first. Objects returned to a pool
// without cleanup carry stale data, so the next Get receives dirty state.
func poolResetBeforePutAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "poolresetbeforeput",
		Doc:      "flags sync.Pool.Put without Reset; objects returned to a pool without cleanup carry stale data into the next Get",
		Run:      runPoolResetBeforePut,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	}
}

// cleanupMethodNames lists method names that count as resetting an object
// before returning it to a sync.Pool.
var cleanupMethodNames = map[string]bool{ //nolint:gochecknoglobals // package-internal lookup table, not mutable state
	"Reset":    true,
	"Truncate": true,
}

func runPoolResetBeforePut(pass *analysis.Pass) (any, error) {
	ssaResult, castOK := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	if !castOK {
		return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
	}

	for _, fn := range ssaResult.SrcFuncs {
		checkFunctionForPoolPut(pass, fn)
	}

	return nil, nil //nolint:nilnil // analysis.Analyzer contract requires (nil, nil) for no results
}

// checkFunctionForPoolPut scans a single SSA function for (*sync.Pool).Put
// calls where the object being put hasn't been cleaned up first.
func checkFunctionForPoolPut(pass *analysis.Pass, fn *ssa.Function) {
	for _, block := range fn.Blocks {
		for instrIdx, instr := range block.Instrs {
			call, isCall := instr.(*ssa.Call)
			if !isCall {
				continue
			}

			if !isPoolPutCall(call) {
				continue
			}

			// arg[0] is the pool receiver, arg[1] is the object being put.
			if len(call.Call.Args) < poolPutMinArgs {
				continue
			}

			// Pool.Put takes any, so SSA wraps the concrete value in
			// MakeInterface. Unwrap to get the underlying typed value that
			// Reset/Truncate would be called on.
			putValue := unwrapMakeInterface(call.Call.Args[1])

			// A freshly allocated object (new(T) or &T{}) has no stale data.
			if isFreshAllocation(putValue) {
				continue
			}

			if hasCleanupBeforePut(block, instrIdx, putValue) {
				continue
			}

			pass.Reportf(
				call.Pos(),
				"sync.Pool.Put without Reset; reset the object before returning it to the pool to avoid stale data",
			)
		}
	}
}

// unwrapMakeInterface strips the MakeInterface wrapper that SSA inserts when
// a concrete value is passed to an interface parameter (like Pool.Put's any).
// Returns the underlying concrete value, or the original if it isn't wrapped.
func unwrapMakeInterface(val ssa.Value) ssa.Value {
	if mi, isMI := val.(*ssa.MakeInterface); isMI {
		return mi.X
	}

	return val
}

// isFreshAllocation reports whether val is a newly allocated object that
// has never been used, so it carries no stale data. Covers new(T) and &T{}.
func isFreshAllocation(val ssa.Value) bool {
	_, isAlloc := val.(*ssa.Alloc)

	return isAlloc
}

// isPoolPutCall reports whether call is a (*sync.Pool).Put invocation.
func isPoolPutCall(call *ssa.Call) bool {
	callee := call.Call.StaticCallee()
	if callee == nil {
		return false
	}

	if callee.Name() != "Put" {
		return false
	}

	sig := callee.Signature

	recv := sig.Recv()
	if recv == nil {
		return false
	}

	return isSyncPoolType(recv.Type())
}

// isSyncPoolType reports whether typ is *sync.Pool.
func isSyncPoolType(typ types.Type) bool {
	ptr, isPtr := typ.(*types.Pointer)
	if !isPtr {
		return false
	}

	named, isNamed := ptr.Elem().(*types.Named)
	if !isNamed {
		return false
	}

	obj := named.Obj()

	return obj.Pkg() != nil && obj.Pkg().Path() == syncPkgPath && obj.Name() == "Pool"
}

// hasCleanupBeforePut checks whether putValue has been cleaned up before the
// Put call. Walks backwards through the Put's basic block first, then checks
// all dominating blocks. SSA can split blocks at panic points (type assertions,
// bounds checks), so the cleanup call may be in a predecessor block.
func hasCleanupBeforePut(block *ssa.BasicBlock, putIdx int, putValue ssa.Value) bool {
	// First check the same block, walking backwards from the Put.
	if blockHasCleanup(block.Instrs[:putIdx], putValue) {
		return true
	}

	// Walk dominating blocks. A dominator executed before this block on
	// every path, so any cleanup there precedes the Put.
	for dominator := block.Idom(); dominator != nil; dominator = dominator.Idom() {
		if blockHasCleanup(dominator.Instrs, putValue) {
			return true
		}
	}

	return false
}

// blockHasCleanup scans instructions for a cleanup call (Reset, Truncate) or
// a Store instruction targeting putValue.
func blockHasCleanup(instrs []ssa.Instruction, putValue ssa.Value) bool {
	for _, instr := range instrs {
		// Check for field zeroing via Store targeting the put value.
		if store, isStore := instr.(*ssa.Store); isStore {
			if store.Addr == putValue {
				return true
			}
		}

		call, isCall := instr.(*ssa.Call)
		if !isCall {
			continue
		}

		callee := call.Call.StaticCallee()
		if callee == nil {
			continue
		}

		if !cleanupMethodNames[callee.Name()] {
			continue
		}

		// The cleanup call's receiver (arg[0]) must match the value being put.
		if len(call.Call.Args) == 0 {
			continue
		}

		if call.Call.Args[0] == putValue {
			return true
		}
	}

	return false
}
