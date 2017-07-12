package main

// A progress-enhanced version of cp that shows progress while copying.
import (
    "os"
    "io"
)

var copysize int64 = 4096

// Copy file contents from source to destination.
func copyFile(src, dst string, progress chan int64) (err error) {
    in, err := os.Open(src)
    if err != nil {
        return
    }
    defer in.Close()
    out, err := os.Create(dst)
    if err != nil {
        return
    }
    // Error handling
    defer func() {
        cerr := out.Close()
        if err == nil {
            err = cerr
        }
    }()
    for {
        var bytes int64 = 0
        bytes, err = io.CopyN(out, in, copysize)
        if err != nil {
            if err == io.EOF {
                progress <- 0
                break
            } else {
                return
            }
        }
        // Report progress
        progress <- bytes
    }
    err = out.Sync()
    return
}

func main() {
    if len(os.Args) < 3 {
        os.Stderr.WriteString("Usage: pcp [options] <source> <destination>\n")
        os.Exit(1)
    }
    source := os.Args[1]
    dest := os.Args[2]

    // A channel for comms with the copying goroutine
    progress := make(chan int64, 1)

    go func() {
        err := copyFile(source, dest, progress)
        if err != nil {
            panic(err)
        }
    }()

    for {
        bytes_copied := <-progress
        println(bytes_copied, "bytes copied")
        if bytes_copied == 0 {
            break
        }
    }

    os.Exit(0)
}
