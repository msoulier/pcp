package main

// A progress-enhanced version of cp that shows progress while copying.
// TODO:
// - add estimated time to completion
// - add -R switch
// - tune progress_freq and resolution for input file size
import (
    "os"
    "io"
    "fmt"
    "math"
    "path"
    mlib "github.com/msoulier/mlib"
    "time"
)

var copysize int64 = 4096
var progress_freq = 1000

// Copy file contents from source to destination.
func copyFile(src, dst string, progress chan int64) (err error) {
    var bytes_copied int64 = 0
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
    i := 0
    for {
        var bytes int64 = 0
        bytes, err = io.CopyN(out, in, copysize)
        if err != nil {
            if err == io.EOF {
                if bytes_copied > 0 {
                    progress <- bytes_copied
                }
                progress <- 0
                break
            } else {
                return
            }
        }
        bytes_copied += bytes
        // Report progress at regular intervals.
        i++
        if i % progress_freq == 0 {
            progress <- bytes_copied
            bytes_copied = 0
        }
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
    var source_size int64 = 0

    // If dest is a directory, add the name of the file to it.
    if stat, err := os.Stat(dest); err == nil && stat.IsDir() {
        // dest is a directory
        source_name := path.Base(source)
        dest = path.Join(dest, source_name)
    }
    // If it doesn't exist, we'll create it as a file. This is standard cp behaviour.

    // stat the source file to get its size
    if stat, err := os.Stat(source); err != nil {
        panic(err)
    } else {
        source_size = stat.Size()
    }

    // A channel for comms with the copying goroutine
    progress := make(chan int64, 1)

    go func() {
        err := copyFile(source, dest, progress)
        if err != nil {
            panic(err)
        }
    }()

    oldTime := time.Now()
    for {
        copied := <-progress
        bytes_copied += copied
        percent := (float64(bytes_copied) / float64(source_size)) * 100

        timeDiff := time.Since(oldTime)
        oldTime = oldTime.Add(timeDiff)
        rate := int64(float64(copied) / timeDiff.Seconds())

        fmt.Printf("\r                                        \r")
        fmt.Printf("progress: %s copied: %d%% - %s/s                          ",
            mlib.Bytes2human(bytes_copied),
            int64(math.Floor(percent)),
            mlib.Bytes2human(rate))
        if copied == 0 {
            break
        }
    }
    fmt.Printf("\ndone\n")

    os.Exit(0)
}
