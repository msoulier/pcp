package main

// A progress-enhanced version of cp that shows progress while copying.
// TODO:
// - add estimated time to completion
// - add -R switch
// - tune progress_freq and resolution for input file size
import (
    "os"
    "io"
    "io/ioutil"
    "fmt"
    "math"
    "path"
    "path/filepath"
    mlib "github.com/msoulier/mlib"
    "time"
)

var copysize int64 = 4096
var progress_freq = 1000
var rate_freq = 50

// Copied from Roland Singer [roland.singer@desertbit.com].

// copyFile copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file. The file mode will be copied from the source and
// the copied data is synced/flushed to stable storage.
func copyFile(src, dst string, progress chan int64) (err error) {
    var bytes_copied int64 = 0
    in, err := os.Open(src)
    if err != nil {
        return err
    }
    defer in.Close()
    out, err := os.Create(dst)
    if err != nil {
        return err
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
                return err
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
    // FIXME: make conditional on a command-line option
    //err = out.Sync()
    return nil
}

// Copied from Roland Singer [roland.singer@desertbit.com].

// copyDir recursively copies a directory tree, attempting to preserve
// permissions.
// Source directory must exist, destination directory must *not* exist.
// Symlinks are ignored and skipped.
func copyDir(src string, dst string, progress chan int64) (err error) {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		return fmt.Errorf("destination already exists")
	}

	err = os.MkdirAll(dst, si.Mode())
	if err != nil {
		return err
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = copyDir(srcPath, dstPath, progress)
			if err != nil {
				return
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			err = copyFile(srcPath, dstPath, progress)
			if err != nil {
				return err
			}
		}
	}
	return nil
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
    var dircopy bool = false

    // Start and end time for the overall completion of the operation.
    start_time := time.Now()


    // If dest is a directory, add the name of the file to it.
    if stat, err := os.Stat(dest); err == nil && stat.IsDir() {
        // dest is a directory
        source_name := path.Base(source)
        dest = path.Join(dest, source_name)
    }
    // If it doesn't exist, we'll create it as a file. This is standard cp
    // behaviour.
    // FIXME: get confirmation if we're overwriting something

    // stat the source file to get its size
    // If there is one source and one dest, check if the source is a
    // directory.
    // FIXME: allow multiple source files
    if stat, err := os.Stat(source); err != nil {
        panic(err)
    } else {
        source_size = stat.Size()
        if stat.IsDir() {
            dircopy = true
        }
    }

    // A channel for comms with the copying goroutine
    progress := make(chan int64, 1)

    go func() {
        var err error = nil
        if dircopy {
            err = copyDir(source, dest, progress)
        } else {
            err = copyFile(source, dest, progress)
        }
        if err != nil {
            panic(err)
        }
    }()

    oldTime := time.Now()
    i := 0
    rate := int64(0)
    time_remaining := time.Duration(0)
    for {
        copied := <-progress
        bytes_copied += copied
        percent := (float64(bytes_copied) / float64(source_size)) * 100
        remaining_bytes := source_size - bytes_copied

        timeDiff := time.Since(oldTime)
        oldTime = oldTime.Add(timeDiff)
        // Only recompute the rate every rate_freq iterations, just to buffer
        // the updates to something readable.
        if i++; i % rate_freq == 0 || rate == 0 {
            rate = int64(float64(copied) / timeDiff.Seconds())
            if rate != 0 {
                time_remaining = time.Duration( float64(remaining_bytes) / float64(rate) ) * time.Second
            }
        }

        fmt.Printf("\r                                                                                \r")
        // FIXME: leave rate and time remaining blank until they're non-zero
        fmt.Printf("%s progress: %7s copied: %3d%% - %7s/s - %s remaining   ",
            source,
            mlib.Bytes2human(bytes_copied),
            int64(math.Floor(percent)),
            mlib.Bytes2human(rate),
            time_remaining)
        if copied == 0 {
            break
        }
    }
    operation_duration := time.Since(start_time)
    fmt.Printf("operation took %s\n", operation_duration)

    os.Exit(0)
}
