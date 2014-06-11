## quick doc

This `main` package defines the wrapper for creating an LS/F command line tool.

### build and try

You can simply `cd` here and try:

    go build

    ./lsf

### what works as of 06/11/2014

* lsf init
* lsf init -f
* lsf stream add
* lsf stream update
* lsf stream [list]  // try verbose flag

### what it will do

LS/F will create a .lsf/ folder in your $HOME directory in addition to and .lsf/
for the local LS/F environment. 