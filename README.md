# supervisor-go
A service that manages service lifetimes.

A supervisor is essentially a more capable errgroup. It monitors a set
of running services, and restarts them if they fail.
The supervisor keeps track of the status of each service and reports any
status changes to listeners via a callback.
