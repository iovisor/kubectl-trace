# bpftrace

This tracer is the default and is treated special, some of the command line
options apply only to bpftrace (such as -e).

# Generic tracers

kubectl-trace supports arbitrary tracers, so long as they adhere to the
existing interface for system (node-level) or process (pod-level) tracing.

If submitting new tracers to the project, they should meet the following
criteria:

- It should adhere to either the system or process tracing interfaces we already have (or both)
- It have broad appeal / will it be useful to a wide audience
- It should not bloat the size of our trace runner image
