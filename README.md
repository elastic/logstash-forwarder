##`LS/F`
LogStash/Forwarder. 

##`stat`

    star date:         july 02 2014 
    
    system:            wip 
    test-suite:        todo 
    documentation:     todo
    
    lsf command:       wip partial functionality
    
    -- configuration & management
    lsf init           ok   adhoc tested
    
    lsf stream:        ok   adhoc tested
    lsf stream add:    ok   adhoc tested
    lsf stream update: ok   adhoc tested
    lsf stream remove: ok   adhoc tested

    lsf remote:        wip  adhoc tested
    lsf stream add:    wip  todo certificates; adhoc tested
    lsf stream update: wip  todo certificates; adhoc tested
    lsf stream remove: ok   adhoc tested
    
    -- operation 
    lsf track:         wip  todo asset snapshot sysdoc;trunc mode; tracking ok adhoc tested
    lsf tail:          todo 
    lsf journal:       todo 
    lsf forward:       todo
    
    -- tools
    lsf migrate:       todo
    lsf monitor:       todo
    lsf gc:            todo

##`Go compliance level`
Codebase is built & tested with **Go1.3**.

##`build`
Std. Go build process. Install in a directory under `$GOPATH/src/`

(*Note the following assumes a \*nix environment.*)

To build the `lsf` command line tool:

    cd <install dir>/lsf
    go build

Above will create the `lsf` executable binary in `<install dir>/lsf`. 

You can freely move this binary to a directory pointed to by your system's `PATH` or can add `<install dir>/lsf` to your `PATH`, as you prefer.

#`overview`

`lsf` command line drew inspiration from `git` user interface. The analog of a `git` *repository* is a *portal* ["***port***"]. Typically you will interact with `LS/F` via the command line, but you can also do so from any `Go` application.

/Aside: "Git?!! That's complicated!" 

**Here is the transition path for existing users**:

    ± lsf init
    ± lsf migrate -config <path to your existing logstash-forwarder conf file>
    ± lsf forward --all-streams

***important***: ***LS/F does not require, nor should it ever be used, as root***. [07/02/2014: wip code does NOT check this! ]

##`port`
A `LS/F` ***port*** is a streaming end-point with associated remote-port peers. The canonical remote-port peer is a `LogStash` server. A port can track and stream to its peer 1 or more logical streams ["***logstream***"].

You can define multiple such ports in your system. Each must reside in a unique path in your filesystem (just like a git repository). Of course you can simply just have 1.



All `lsf` commands (and processes) work ["lsf working directory"] in the context of a given port.

Ok, so let's create our first ***port***:

    # create a LS/F port - cd to path
    ± cd ~/playground/testing
    ± lsf init    
    Initialized LSF environment at /Users/alphazero/playground/testing/.lsf

This will create the initial minimal `.lsf` directory in the working directory. 

A port can be reinitialized either by deleting the `.lsf` directory and running the `lsf init` command, or using `lsf` directly. (***Note that this will remove all definitions and stored objects***. But you can always simply `cp -r .lsf <backup dir>` a port.)

    # start over
    ± rm -rf .lsf
    ± lsf init
    
    # or by forcing a re-init
    ± lsf init -f
    Re-Initialize LSF environment at /Users/alphazero/playground/testing/.lsf

If you are curious, try:

    ± ls -la .lsf
    total 8
    drwxr-xr-x  4 alphazero  staff  136 Jul  2 21:06 .
    drwxr-xr-x  3 alphazero  staff  102 Jul  2 21:04 ..
    -rw-r--r--  1 alphazero  staff   52 Jul  2 21:04 SYSTEM    
    
    ± cat .lsf/SYSTEM
    create-time: 2014-07-02 21:08:47.333131046 -0400 EDT

##`logstream`
Each port can be configured to track and stream a multiplicity of filesystem objects (typically your log files). These logical streams are named and these names are used with applicable `lsf` commands to act on the ***logstream***.

Streams are defined & managed with the `lsf stream [sub-command]`s. 

(Note that the most of `lsf` commands are exclusive operations and `LS/F` will not allow concurrent execution of the **same** operation in the **same** port to insure system integrity and correct handling of your data. (Concurrent ops via distinct ports, even on the same set of filesystem objects, are perfectly ok. For example, you can stream the same set of log files to different remote or local ports.)

    # list a port's streams
    # if this is a new port, then it will simply return.
    # otherwise, the names of the port's defined streams are listed.
    # use the verbose flag to display stream details.

    ± lsf stream [-v | -verbose]
    
Let's ***define a logstream***:

    ± lsf stream -s <stream-name> -p <basepath of tracked objects> -n <glob pattern> -m <mode>
    
    # example:
    # you capture the click events of your server in a series of log files
    # that are rotated. Let's say the log files are in /var/logs/my-app/ 
    # and named per pattern click.log[.*]
   
    ± lsf stream add -s clickevents -p /var/logs/my-app -n "apache2.log*" -m rotation
    
    # lsf is quiet when successful. confirm:
    ± lsf stream -v 
    logstream clickevents /var/logs/my-app rotation apache2.log* map[]

Did we fat finger something? Let's ***update a logstream***:

    # ooops. we got the file pattern wrong: let's correct it:
    ± lsf stream update -s clickevents -n "click.log*"
    
    # confirm:
    ± lsf stream -v
    logstream clickevents /var/logs/my-app rotation click.log* map[]
        
If you are curious, try:

    ± ls -la .lsf/stream
    total 0
    drwxr-xr-x  3 alphazero  staff  102 Jul  2 21:06 .
    drwxr-xr-x  4 alphazero  staff  136 Jul  2 21:06 ..
    drwxr-xr-x  3 alphazero  staff  102 Jul  2 21:06 clickevents    
    
    ± ls -la .lsf/stream/clickevents
    total 8
    drwxr-xr-x  3 alphazero  staff  102 Jul  2 21:11 .
    drwxr-xr-x  3 alphazero  staff  102 Jul  2 21:10 ..
    -rw-r--r--  1 alphazero  staff   89 Jul  2 21:11 STREAM

    ± cat .lsf/stream/clickevents/STREAM
    basepath: /var/logs/my-app
    pattern: click.log*
    journal-model: rotation
    id: clickevents
    
Yep. It's just a text file. 

You can edit it -- and pretty much everything else -- directly, if you feel like it! (Note: insure the port is not active via `lsf monitor`. [07/02/2014: not yet implemented.])

##`remote`
A *remote port* ["***remote***"] is the peer end-point that consumes the stream generated by a (local) ***port***. The canonical and intended remote peer is of course `LogStash`.

So what defines a remote port? No surprises here: remote IP address, a logical name, and certificates. [07/02/2014: certs coming up soon.]


Alright then, let's ***define a remote*** port:
 
    # 'ls-cluster' is just a logical name. ops on this remote use this logical name.
    
    ± lsf remote add -r ls-cluster -h 122.140.201.1 -p 6333 
    
    # let's confirm:
    ± ls remote -v
    port ls-cluster remote 122.140.201.1:6333

[TODO: remote update ; remote remove]

##`track`
The most basic capability of `LS/F` is the tracking of filesystem objects that match the specification of a `stream`. The `lsf track` command provides the command line interface to this capability. `track` generates an event log and a filesystem snapshot of the relavant filesystem objects. 

Invoking `lsf track` with the help option will display the gory details:

    ± lsf track -h
    Usage of track:
    -G=false: command applies globally
    -N=0: max size of fs object cache
    -T=0: max age of objects in fs object cache
    -f=1: report frequency - n / sec (e.g. 1000 1/ms)
    -frequency=1: report frequency - n / sec (e.g. 1000 1/ms)
    -global=false: command applies globally
    -max-age=0: max age of objects in fs object cache
    -max-size=0: max size of fs object cache
    -s="": unique identifier for stream
    -stream-id="": unique identifier for stream
    
Try defining a stream that matches your current `logstash-forwarder` settings and try the minimal form of this command.

    # assuming that we have (per above) defined a stream
    # with id 'clickevents'
    
    # we minimally need to identify the logstream to track
    # The '-N' option caps the numbers of FS objects tracked.
    # For logstreams that are rotated (such as apache2.log.n)
    # pick a value that is an integral multiple of the max extension
    # value. For example, here we use 16 as our hypothetical logger
    # rotates files up to *.log.15.
    # For loggers that truncate in place, you will want to use
    # the T | max-age option.
    
    lsf track -s clickevents -N 16
    
The above will [as of 08/02/2] emit to std. out, a set of FS events observed by the tracker, and, the updated snapshot of objects that map to the stream's spec. Give it a try.

[tbc]