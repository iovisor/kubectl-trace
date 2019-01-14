# kubectl trace

<!-- toc -->

`kubectl trace` is a kubectl plugin that allows you to schedule the execution
of [bpftrace](https://github.com/iovisor/bpftrace) programs in your Kubernetes cluster.

![Screenshot showing the read.bt program for kubectl-trace](docs/img/intro.png)

## Installation

```
go get -u github.com/iovisor/kubectl-trace/cmd/kubectl-trace
```

This will download and compile `kubectl-trace` so that you can use it as a kubectl plugin with `kubectl trace`

## Usage

You don't need to setup anything on your cluster before using it, please don't use it already
on a production system, just because this isn't yet 100% ready.

**Run a program from string literal:**

In this  case we are running a program that probes a tracepoint
on the node `ip-180-12-0-152.ec2.internal`.

```
kubectl trace run ip-180-12-0-152.ec2.internal -e "tracepoint:syscalls:sys_enter_* { @[probe] = count(); }"
```


**Run a program from file:**

Here we run a program named `read.bt` against the node `ip-180-12-0-152.ec2.internal`

```
kubectl trace run ip-180-12-0-152.ec2.internal -f read.bt
```

**Run a program against a Pod**

![Screenshot showing the read.bt program for kubectl-trace](docs/img/pod.png)

That pod has a Go program in it that is at `/caturday`, that program has a function called `main.counterValue` in it that returns an integer
every time it is called.

The purpose of this program is to load an `uretprobe` on the `/caturday` binary so that every time the `main.counterValue` function is called
we get the return value out.

Since `kubectl trace` for pods is just an helper to resolve the context of a container's Pod, you will always be in the root namespaces
but in this case you will have a variable `$container_pid` containing the pid of the root process in that container on the root pid namespace.

What you do then is that you get the `/caturday` binary via `/proc/$container_pid/exe`, like this:

```
kubectl trace run -e 'uretprobe:/proc/$container_pid/exe:"main.counterValue" { printf("%d\n", retval) }' pod/caturday-566d99889-8glv9 -a -n caturday
```

### Running against a Pod vs against a Node

In general, you run kprobes/kretprobes, tracepoints, software, hardware and profile events against nodes using the `node/node-name` syntax or just use the
node name, node is the default.

When you want to actually probe an userspace program with an uprobe/uretprobe or use an user-level static tracepoint (usdt) your best
bet is to run it against a pod using the `pod/pod-name` syntax.

It's always important to remember that running a program against a pod, as of now, is just a facilitator to find the process id for the binary you want to probe
on the root process namespace.

You could do the same thing when running in a Node by knowing the pid of your process yourself after entering in the node via another medium, e.g: ssh.

So, running against a pod **doesn't mean** that your bpftrace program will be contained in that pod but just that it will pass to your program some
knowledge of the context of a container, in this case only the root process id is supported via the `$container_pid` variable.


### More bpftrace programs

Need more programs? Look [here](https://github.com/iovisor/bpftrace/tree/master/tools).

## Status of the project

:trophy: All the MVP goals are done!

To consider this project (ready) the goals are:

- [x] basic program run and attach
- [x] list command to list running traces - command: `kubectl trace get`
- [x] delete running traces
- [x] run without attach
- [x] attach command to attach only - command: `kubectl trace attach <program>`
- [x] allow sending signals (probably requires a TTY), so that bpftrace commands can be notified to stop by the user before deletion and give back results


**More things after the MVP:**

<i>The stuff here had been implemented - YaY</i>

<strike>The program is now limited to run programs only on your nodes but the idea is to have the ability to attach only to the user namespace of a pod, like:

```
kubectl trace run pod/<pod-name> -f read.bt
```

And even on a specific container

```
kubectl trace run pod/<pod-name> -c <container> f read.bt
```

So I would say, the next thing is to run bpftrace programs at a pod scope other than at node scope.</strike>

**bpftrace work**

I also plan to contribute some IO functions to bpftrace to send data to a backend database like InfluxDB instead of only stdout
because that would enable having things like graphs showing 

## Contributing

Please just do it, this is MIT licensed so no reason not to!
