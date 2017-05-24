# REDCap file system

A file system interface to an electronic data capture system. This implementation uses file system in user 
space (fuse) to map file system operations to read operations towards a REDCap database.

Once you start the program you can download a REDCap instrument of your choice by creating a file name in the mounted
directory. The program will fill your file with the data exported from REDCap. Name your file with a specific extension
to get a particular encoding. Currently the program supports JSON, CSV and Excels xlsx.

The program creates a connection to REDCap using a token that has to be created in REDCap. The REDCap API will be used 
with this token to request data. Because the token is sufficient to create a connection to REDCap its value is stored
encrypted in the current directory. Running this program again will ask for a pass phrase to un-encrypt the token.

### Build

go build *.go

If you created the connection previously you need to remove the mount point again before you can do it a second time for the same directory:
```
  /bin/fusermount -u test
```

Afterwards run the program again:
```
  ./redcapfs /tmp/here
```

### Requirements on MacOS

   - install FUSE for macOS: https://osxfuse.github.io
   - Initially run: /Library/Filesystems/osxfuse.fs/Contents/Resources/load_osxfuse (restart might work as well)
   - to un-mount a directory use umount <directory>

### Example Session

Initially create a link to your REDCap database. This is done in REDCap using tokens that depend on both the user account and the project. Only with such a token the interface will be able to connect to the database. Tokens are stored by the application in a local file (encrypted). Every time you start the program you will be asked for a password to that store.

```
> ./redcapfs --help
Usage of ./redcapfs:
  -addToken string
    	add a <REDCap token>
  -clearAllToken
    	remove stored token
  -debug
    	print debugging messages.
  -setREDCapURL string
    	set the REDCap URL (default "https://abcd-rc.ucsd.edu/redcap/api/")
  -showToken
    	show existing token
>
>
> /Library/Filesystems/osxfuse.fs/Contents/Resources/load_osxfuse
> ./redcapfs /tmp/EDC
This is a secured access. Provide your pass phrase: 
Mounted!
```
There will be two files in the created directory connection to the database.

```
> ls
DataDictionary.json	EventMapping.json
```

The EventMapping.json file contains the names of forms used in the study. Now we can create a new file in our directory with the name of the form in question:

```
> touch screener.csv
```

It depends on the speed of your connection to the EDC but after a couple of seconds the new file will fill with the data for that instrument. There will also be a second file that contains the data dictionory for that form.

```
> ls -l
total 17656
-rw-r--r--  1 hauke  staff  8569835 May 24 13:58 DataDictionary.json
-rw-r--r--  1 hauke  staff    20978 May 24 13:58 EventMapping.json
-rw-r--r--  1 hauke  staff   367411 May 24 14:03 screener.csv
-rw-r--r--  1 hauke  staff    72963 May 24 14:03 screener_datadictionary.csv
```
