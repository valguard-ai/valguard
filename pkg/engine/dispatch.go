package engine

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/valguard-ai/valguard/pkg/validators"
)

// parallelRuleThreshold determines when RunAttached starts goroutines for
// rule dispatch. Each goroutine costs ~1 µs scheduling overhead on Linux,
// and most validators in our pack complete in single-digit microseconds
// after the recent optimisations. The shallow JSON cache means duplicated
// parse work is no longer the dominant cost we used to amortise across
// goroutines, so we go sequential up to a generous bound. Heavy packs
// (PII / secrets / RAG with regex sweeps) still benefit from parallelism.
const parallelRuleThreshold = 16

// maxParallelRules caps concurrent validator goroutines and the pooled
// dispatch semaphore buffer (RunAttached clamps len(instances) to this).
const maxParallelRules = 32

type FailAction struct {
	Block  bool
	Reask  bool
	Warn   bool
	Failed []validators.Result
	Names  []string
}

type RunOutput struct {
	Results     []validators.Result
	Blocked     bool
	ShouldReask bool
	Failures    []ValidationFailure

	// resultsBuf is the pool-owned backing slice for Results. Callers
	// reach it through ReleaseRun; it stays nil for the single-rule
	// fast path which uses an inline array (no pool round-trip).
	resultsBuf *[]validators.Result
	// failuresBuf mirrors resultsBuf for pooled ValidationFailure slices.
	failuresBuf *[]ValidationFailure
}

// resultsPool keeps Result slices alive across calls. The pool stores
// *[]validators.Result rather than []validators.Result so that
// sync.Pool.Put does not need to box the slice header into an interface{}
// (single-word pointers fit directly into iface storage without a heap
// allocation).
var resultsPool = sync.Pool{
	New: func() any { s := make([]validators.Result, 0, 16); return &s },
}

var failuresPool = sync.Pool{
	New: func() any { s := make([]ValidationFailure, 0, 4); return &s },
}

func acquireFailures() (*[]ValidationFailure, []ValidationFailure) {
	buf := failuresPool.Get().(*[]ValidationFailure)
	s := (*buf)[:0]
	*buf = s
	return buf, s
}

// acquireResults returns a slice of length n plus the pool pointer
// owning the backing array. Callers MUST call ReleaseRun (or
// resultsPool.Put) with the same pointer to recycle the slice.
func acquireResults(n int) (slice []validators.Result, buf *[]validators.Result) {
	buf = resultsPool.Get().(*[]validators.Result)
	s := *buf
	if cap(s) < n {
		s = make([]validators.Result, n)
	} else {
		s = s[:n]
		for i := range s {
			s[i] = validators.Result{}
		}
	}
	*buf = s
	return s, buf
}

// ReleaseRun returns the pooled backing slice to the pool. Safe to call
// at most once per RunOutput. Callers that need to retain Results past
// the next validator run must skip this call.
func ReleaseRun(out RunOutput) {
	if out.failuresBuf != nil {
		*out.failuresBuf = (*out.failuresBuf)[:0]
		failuresPool.Put(out.failuresBuf)
	}
	if out.resultsBuf == nil {
		return
	}
	*out.resultsBuf = (*out.resultsBuf)[:0]
	resultsPool.Put(out.resultsBuf)
}

type ValidationFailure struct {
	ValidatorName string
	ValidatorID   string
	Message       string
	OnFail        string
	Severity      string
	Details       map[string]any
}

// RunOptions controls validator dispatch behaviour.
type RunOptions struct {
	// StopOnFirstEnforce skips remaining rules after the first block/reask failure.
	// Used for live inbound prompts; dashboard preview runs all enabled rules.
	StopOnFirstEnforce bool
	// PreRun, when set, may skip execution and supply a result (plan gating, etc.).
	PreRun func(inst Instance) (skip bool, result validators.Result)
}

func Evaluate(instances []Instance, results []validators.Result) RunOutput {
	return evaluate(instances, results, nil)
}

func evaluate(instances []Instance, results []validators.Result, buf *[]validators.Result) RunOutput {
	var resultsOut []validators.Result
	if len(results) == len(instances) {
		resultsOut = results
	} else {
		n := len(results)
		if n > len(instances) {
			n = len(instances)
		}
		resultsOut = results[:n:n]
	}
	out := RunOutput{
		Results:    resultsOut,
		resultsBuf: buf,
	}
	var failuresBuf *[]ValidationFailure
	for i, inst := range instances {
		if i >= len(results) {
			continue
		}
		r := results[i]
		if !inst.Enabled || r.Status != "fail" {
			continue
		}
		if out.Failures == nil {
			failuresBuf, out.Failures = acquireFailures()
		}
		out.Failures = append(out.Failures, ValidationFailure{
			ValidatorName: inst.Name,
			ValidatorID:   inst.ID,
			Message:       r.Message,
			OnFail:        inst.OnFail,
			Severity:      r.Severity,
			Details:       r.Details,
		})
		switch inst.OnFail {
		case "block":
			out.Blocked = true
		case "reask":
			out.ShouldReask = true
		case "pass", "log":
			// log: per-rule shadow — record failure, pass response through
		default:
			// warn and unknown: do not block/reask
		}
	}
	out.failuresBuf = failuresBuf
	return out
}

// Run validates output with AttachRunLite (OSS CLI / tests).
// ValidatorRegistry resolves catalog rules by name (proxy and OSS registries).
type ValidatorRegistry interface {
	Get(name string) (validators.Validator, bool)
}

func Run(ctx context.Context, reg ValidatorRegistry, instances []Instance, output string, meta validators.RunMeta) RunOutput {
	return RunForPlan(ctx, reg, instances, output, meta, RunOptions{})
}

// RunForPlan attaches run meta with AttachRunLite then dispatches validators.
func RunForPlan(ctx context.Context, reg ValidatorRegistry, instances []Instance, output string, meta validators.RunMeta, opts RunOptions) RunOutput {
	ctx = validators.AttachRunLite(ctx, meta, output)
	return RunAttached(ctx, reg, instances, output, opts)
}

// RunAttached dispatches validators; ctx must already carry run meta/output cache.
func RunAttached(ctx context.Context, reg ValidatorRegistry, instances []Instance, output string, opts RunOptions) RunOutput {
	results, buf := acquireResults(len(instances))

	if len(instances) == 1 {
		results[0] = runOne(ctx, reg, instances[0], output, opts)
		return evaluate(instances, results, buf)
	}

	// Sequential path for small packs avoids goroutine + channel overhead
	// entirely (no batch allocation either). Early-stop live paths also
	// stay sequential so we can bail out after the first enforce failure.
	if len(instances) <= parallelRuleThreshold || opts.StopOnFirstEnforce {
		ran := runSequential(ctx, instances, results, func(i int) {
			results[i] = runOne(ctx, reg, instances[i], output, opts)
		}, opts)
		return evaluate(instances[:ran], results[:ran], buf)
	}

	batch := acquireRunBatch()
	defer releaseRunBatch(batch)
	batch.ctx = ctx
	batch.reg = reg
	batch.inst = instances
	batch.output = output
	batch.opts = opts
	batch.results = results
	batch.sem = ensureDispatchSem(batch)

	for i := range instances {
		if ctx.Err() != nil {
			batch.cancelled.Store(true)
			break
		}
		if res, skip := preRunResult(instances[i], opts); skip {
			results[i] = res
			continue
		}
		batch.wg.Add(1)
		batch.sem <- struct{}{}
		go batch.runAtAndRelease(i)
	}
	done := make(chan struct{})
	go func() {
		batch.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		batch.cancelled.Store(true)
		<-done
	}
	return evaluate(instances, results, buf)
}

func runSequential(ctx context.Context, instances []Instance, results []validators.Result, run func(int), opts RunOptions) int {
	ran := len(instances)
	for i := range instances {
		select {
		case <-ctx.Done():
			return i
		default:
		}
		run(i)
		if opts.StopOnFirstEnforce && InstanceWouldEnforce(instances[i], results[i]) {
			ran = i + 1
			break
		}
	}
	return ran
}

func cancelledValidatorResult() validators.Result {
	return validators.Result{Status: "skipped", Severity: "error", Message: "cancelled"}
}

func preRunResult(inst Instance, opts RunOptions) (validators.Result, bool) {
	if opts.PreRun == nil {
		return validators.Result{}, false
	}
	skip, res := opts.PreRun(inst)
	return res, skip
}

func runOne(ctx context.Context, reg ValidatorRegistry, inst Instance, output string, opts RunOptions) validators.Result {
	if inst.ParamsInvalid {
		return validators.Result{
			Status: "fail", Severity: "error", Message: "invalid params JSON",
		}
	}
	if !inst.Enabled {
		return validators.Result{Status: "skipped", Severity: "error", Message: "disabled"}
	}
	if res, skip := preRunResult(inst, opts); skip {
		return res
	}
	if inst.Compiled != nil {
		return inst.Compiled(ctx, output)
	}
	v := inst.Validator
	if v == nil && reg != nil {
		var ok bool
		v, ok = reg.Get(inst.Name)
		if !ok {
			return validators.Result{
				Status: "fail", Severity: "error", Message: "unknown validator: " + inst.Name,
			}
		}
	}
	if v == nil {
		return validators.Result{
			Status: "fail", Severity: "error", Message: "unknown validator: " + inst.Name,
		}
	}
	return v.Run(ctx, output, inst.Params)
}

// runBatch carries per-Run shared state for parallel dispatch.
type runBatch struct {
	ctx       context.Context
	reg       ValidatorRegistry
	inst      []Instance
	output    string
	opts      RunOptions
	results   []validators.Result
	sem       chan struct{}
	wg        sync.WaitGroup
	cancelled atomic.Bool
}

func (b *runBatch) runAtAndRelease(idx int) {
	defer func() {
		<-b.sem
		b.wg.Done()
	}()
	if b.cancelled.Load() {
		b.results[idx] = cancelledValidatorResult()
		return
	}
	b.results[idx] = runOne(b.ctx, b.reg, b.inst[idx], b.output, b.opts)
}

var runBatchPool = sync.Pool{
	New: func() any {
		return &runBatch{sem: make(chan struct{}, maxParallelRules)}
	},
}

func ensureDispatchSem(b *runBatch) chan struct{} {
	if b.sem == nil {
		b.sem = make(chan struct{}, maxParallelRules)
		return b.sem
	}
	for {
		select {
		case <-b.sem:
		default:
			return b.sem
		}
	}
}

func acquireRunBatch() *runBatch {
	return runBatchPool.Get().(*runBatch)
}

func releaseRunBatch(b *runBatch) {
	b.ctx = nil
	b.reg = nil
	b.inst = nil
	b.output = ""
	b.opts = RunOptions{}
	b.results = nil
	runBatchPool.Put(b)
}
