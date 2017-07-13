package main

// A progress-enhanced version of cp that shows progress while copying.
// TODO:
// - add estimated time to completion
// - add more cp-like behaviour
// - add -R switch
import (
    "os"
    "io"
    "fmt"
    "math"
    mlib "github.com/msoulier/mlib"
)

var copysize int64 = 4096
var progress_freq = 1000

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
    var bytes_copied int64 = 0

    // stat the source file to get its size
    source_size, err := mlib.StatfileSize(source)
    if err != nil {
        panic(err)
    }

    // A channel for comms with the copying goroutine
    progress := make(chan int64, 1)

    go func() {
        err := copyFile(source, dest, progress)
        if err != nil {
            panic(err)
        }
    }()

    i := 0
    fmt.Printf("\n")
    for {
        copied := <-progress
        bytes_copied += copied
        percent := (float64(bytes_copied) / float64(source_size)) * 100
        if i % progress_freq == 0 || copied == 0 {
            fmt.Printf("\r                                        \r")
            fmt.Printf("progress: %d%%                           ", int64(math.Floor(percent)))
        }
        i++
        if copied == 0 {
            break
        }
    }
    fmt.Printf("\ndone\n")

    os.Exit(0)
}
