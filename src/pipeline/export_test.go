package pipeline

import "context"

// RunAnalyseWorker exposes the unexported runAnalyseWorker for pipeline_test.
var RunAnalyseWorker = func(p *Pipeline, ctx context.Context, workerDone chan<- struct{}) {
	p.runAnalyseWorker(ctx, workerDone)
}

// AnalyseQueue exposes the unexported analyseQueue channel for test injection.
var AnalyseQueue = func(p *Pipeline) chan CaptureResult {
	return p.analyseQueue
}
