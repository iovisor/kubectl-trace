# kubectl trace

`kubectl trace` is a kubectl plugin that allows you to schedule the execution
of [bpftrace](https://github.com/iovisor/bpftrace) programs in your Kubernetes cluster.

![Screenshot showing the read.bt program for kubectl-trace](docs/img/intro.png)

## Installation

```
go get -u github.com/fntlnz/kubectl-trace/cmd/kubectl-trace
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

In this case we are running our tracing program in the same pid namespace (`man 7 pid_namespaces`) of the default container of the pod named
`caturday-566d99889-8glv9`. Our trace program will be also sharing the same rootfs with that container.

That pod has a Go program in it that is at `/caturday`, that program has a function called `main.counterValue` in it that returns an integer
every time it is called.

The purpose of this program is to load an `uretprobe` on the `/caturday` binary so that every time thhe `main.counterValue` function is called
we get the return value out.

```
kubectl trace run -e 'uretprobe:/caturday:"main.counterValue" { printf("%d\n", retval) }' pod/caturday-566d99889-8glv9 -a -n caturday
```

**Important note** The fact that the trace programs runs in the same pid namespace and under the same chroot **doesn't mean** that the trace program
is contained in that container, so if you have another caturday binary running from the same image (or ELF binary) in the same machine you will be dumping results from both.

### Running against a Pod vs against a Node

In general, you run kprobes/kretprobes, tracepoints, software, hardware and profile events against nodes using the `node/node-name` syntax or just use the
node name, node is the default.

When you want to actually probe an userspace program with an uprobe/uretprobe or use an user-level stattic tracepoint (usdt) your best
bet is to run it against a pod using the `pod/pod-name` syntax.

It's always important to remember that running a program against a pod, as of now, is just a facilitator to find the binary you want to probe, you are in the same root filesystem so if your binary is in `/mybinary` it's easier to find it from there. You could do the same thing when running in a Node by 
knowing the pid of your binary to get it from the proc filesystem like `/proc/12345/exe` but that would require extra machine access to actually find
the pid. So, running against a pod **doesn't mean** that your bpftrace program will be contained in that pod but just that it will run from the same root filesystem.


### More bpftrace programs

Need more programs? Look [here](https://github.com/iovisor/bpftrace/tree/master/tools)

Some of them will not yet work because we don't attach with a TTY already, sorry for that but good news you can contribute it!


## Contributing

Please just do it, this is MIT licensed so no reason not to!
