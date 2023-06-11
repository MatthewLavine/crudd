# Continuously Running Userland Diagnostics Daemon (CRUDD)

CRUDD allows a user to remotely execute a series of "safe" diagnostics tools via a web interface.

CRUDD's number one invariant is that it will only run pre-vetted commands and does not provide arbitrary remote code execution.

This is loosely based on a workstation dianostics daemon used at Google but shares zero code with it.

## Running CRUDD
### Prerequesites

 - Have Go installed

 - (Optional) Have Docker installed

### Running directly

`$ go run crudd`

Access crudd at `localhost:4901`

### Running in Docker

`$ docker build -t crudd . &&  docker run --rm -it -p 4901:4901/tcp crudd`

Access crudd at `localhost:4901`

### Installing as a Service

Run `install.sh` to:

1) Compile crudd
1) Install crudd to `/usr/bin/crudd`
1) Install `crudd.service` as a Systemd service
1) Enable and start `crudd.service`

Access crudd at `localhost:4901`