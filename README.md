# lzt

eBPF and mTLS experiments

## Socket interception and exchange by supervisor

Demonstrate the use of supervisors to exchange data on existing connection. The demo
 connection uses an echo client and server. Both pause after connecting and before
 any data is exchanged. Currently suspension is handled by voluntary call to
 `syscall.Kill(syscall.GetPid, syscall.SIGSTOP)`, but this shall be replaced by
 eBPF/system call interception.
 Supervisors (`maitred`) are invoked with the process id to monitor (printed by
 `echo-client` and `echo-server` on start-up) and the file descriptor of the newly
 created socket (typically fd #4 on the server and #3 on the client).
 The supervisors exchange PING/PONG messages and then resume the suspended echo
 processes. The PING/PONG can be later replaced by AuthN/AuthZ (e.g., mTLS).

1. Build executables

   ```sh
   go build -o echo-server cmd/echo-server/main.go
   go build -o echo-client cmd/echo-client/main.go
   go build -o maitred cmd/maitred/main.go  
   ```

1. Grant maitred capabilities (PTRACE for pidfd_getfd and KILL for sending SIGCONT)

   ```sh
   $ sudo setcap cap_sys_ptrace,cap_kill+ep ./maitred
   $ getcap ./maitred
   ./maitred = cap_kill,cap_sys_ptrace+ep
   ```

1. Run demo (easier with three terminals: main for echo client and server and two more,
 one for each supervisor)

   ```sh
   $ # main terminal
   $ ./echo-server &
   [1] 497961
   2024/06/13 13:15:15 server (PID 497961) listening on :3333
   $ ./echo-client
   2024/06/13 13:23:51 client (PID 498361) new connection 3 (127.0.0.1:43866 -> 127.0.0.1:3333)
   2024/06/13 13:23:51 process 497961 accepted connection 4 (127.0.0.1:43866 -> 127.0.0.1:3333)
   2024/06/13 13:23:51 process 498361 sending SIGSTOP to self
   2024/06/13 13:23:51 process 497961 sending SIGSTOP to self
   $ # server supervisor terminal
   $ ./maitred -server -fd 4 -pid 497961
   2024/06/13 13:23:59 supervisor (pid 498412) duplicating fd 4 from pid 497961
   2024/06/13 13:23:59 supervisor (pid 498412) hijacked 127.0.0.1:43866 -> 127.0.0.1:3333 from pid 497961
   2024/06/13 13:24:02 supervisor (pid 498412): PING from 498436 on 127.0.0.1:3333 -> 127.0.0.1:43866
   $ # client supervisor terminal
   $ ./maitred -fd 3 -pid 498361
   2024/06/13 13:24:02 supervisor (pid 498436) duplicating fd 3 from pid 498361
   2024/06/13 13:24:02 supervisor (pid 498436) hijacked 127.0.0.1:43866 -> 127.0.0.1:3333 from pid 498361
   2024/06/13 13:24:02 supervisor (pid 498436): PONG from 498412 on 127.0.0.1:43866 -> 127.0.0.1:3333
   $ # meanwhile, on the main terminal
   2024/06/13 13:24:02 process 497961 continuing
   2024/06/13 13:24:02 process 498361 continuing
   2024/06/13 13:24:02 498361 closed connection 3
   2024/06/13 13:24:02 497961 closed connection 4
   ```

1. Kill running processes

   ```sh
   # main terminal
   $ kill -9 497961
   ```