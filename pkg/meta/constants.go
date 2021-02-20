package meta

const (
	// TracePrefix is the prefix to identify objects created by this tool
	TracePrefix = "kubectl-trace-"
	// TraceIDLabelKey is a meta to annotate objects created by this tool
	TraceIDLabelKey = "iovisor.org/kubectl-trace-id"
	// TraceLabelKey is a meta to annotate objects created by this tool
	TraceLabelKey = "iovisor.org/kubectl-trace"
	Label         = "ELK"
	// ObjectNamePrefix is the prefix used for objects created by kubectl-trace
	ObjectNamePrefix = "kubectl-trace-"
)
