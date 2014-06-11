## quick doc

This `main` package defines the wrapper for creating an LS/F command line tool.

### build and try

You can simply `cd` here and try:

    go build

    ./lsf

### sequence to try (assuming your PATH is pointing at the right path)

   # initialize an LS/F port in a given location

   cd <somewhere>
   lsf init

   # list defined streams
   # we have nothing yet so NOP

   lsf stream [-v]

   # let's define a stream
   # try 'lsf stream add -h' for stream-add command's usage options

   lsf stream add -s apache-123 -n "*.log*" -p /var/logs -m rotation

   # now try listing them again

   lsf stream
   lsf stream -v

   # now let's update the stream definition
   # let's change the path to the log files

   lsf stream update -s apache-123 -p /var/log/apache-logs -m rotation

   # and confirm
   # should reflect the updated stream spec
   lsf stream -v

   # what's out there?
   # open and take a look

   open .lsf/


### what works as of 06/11/2014

#### commands

* lsf init
* lsf init -f
* lsf stream add
* lsf stream update
* lsf stream [list]  // try verbose flag

#### features

* FS based concurrent lock service
* FS based hierarchical atomic doc (k/v) service
* Command runner framework and above set of commands to iron things out


### what it will do

LS/F will create a .lsf/ folder in your $HOME directory in addition to and .lsf/
for the local LS/F environment.