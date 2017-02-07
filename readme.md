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


