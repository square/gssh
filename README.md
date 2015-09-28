#### gssh

simple command line to utility to run commands on multiple hosts in parallel

##### Usage

```bash
echo host1 > /tmp/hosts
echo host2 >> /tmp/hosts
gssh -f /tmp/hosts uname -a

#host1:stdout:Linux host1 2.6.32-431.11.2.el6.x86_64 #1 SMP Tue Mar 25 19:59:55 UTC 2014 x86_64 x86_64 x86_64 GNU/Linux
#host2:stdout:Linux host2 2.6.32-431.11.2.el6.x86_64 #1 SMP Tue Mar 25 19:59:55 UTC 2014 x86_64 x86_64 x86_64 GNU/Linux
```

```bash
gssh -r host1..2 uname -a
```

###### Overriding ssh options

Example:

```bash
gssh -f file -- -o ConnectTimeout=30 -o BatchMode=yes id
```

gssh supports streaming support. Useful for tailing logs across multiple machines etc

```bash
gssh -f file -- tail -F /var/log/secure | grep -i Accepted
```

##### Installation

1. Install go
2. `go get -u -v github.com/square/gssh`

##### Development
* We use godep for vendoring and dependency management.
  * `godep restore # restore to last known good set`
* Please run `gofmt` and `golint` before submitting PRs
