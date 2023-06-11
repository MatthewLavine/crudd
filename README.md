# Continuously Running Userland Diagnostics Daemon (CRUDD)

CRUDD allows a user to remotely execute a series of "safe" diagnostics tools via a web interface.

CRUDD's number one invariant is that it will only run pre-vetted commands and does not provide arbitrary remote code execution.

This is loosely based on a workstation dianostics daemon used at Google but shares zero code with it.