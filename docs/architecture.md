# Architecture

Since it is a kubectl plugin, kubectl-trace doesn't require you to install any component directly
to your kubernetes cluster in order to execute your bpftrace programs, however when you point it to
a cluster, it will schedule a temporary job there called `trace-runner` that executes the program.

This figure, shows the general idea:

![Kubectl trace architecture diagram](/docs/img/kubectl-trace-architecture.png)
