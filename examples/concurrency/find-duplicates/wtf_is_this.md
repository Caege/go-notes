# File Size Grouper Using Worker Pool + Single Dispatcher

## Goal

Scan an entire drive recursively, find all `.mp4` files, group them by file size, and print groups that contain more than one file.

This is the first stage of a duplicate file finder.

Files with different sizes cannot be duplicates, so grouping by size dramatically reduces the number of files that need hashing later.

---

## Architecture

```text
Dispatcher
    ↓
Jobs Channel
    ↓
Worker Pool (4 workers)
    ↓
File Size Map
```

### Dispatcher

The main goroutine acts as a scheduler.

Responsibilities:

* Tracks directories waiting to be scanned (`pending`)
* Tracks active directory scan jobs (`active`)
* Sends work to workers
* Receives newly discovered directories
* Determines when traversal is complete

Only the dispatcher modifies:

```go
pending
active
```

This makes termination detection simple.

---

## Worker Pool

A fixed number of workers are created:

```go
workerCount := 4
```

Each worker:

1. Receives a directory path from `jobs`
2. Reads directory contents using:

```go
os.ReadDir()
```

3. Discovers subdirectories
4. Finds `.mp4` files
5. Gets file size
6. Stores file path in the size map
7. Returns newly discovered directories back to the dispatcher

Workers do not schedule new work directly.

Workers only report findings.

---

## Pending Queue

```go
pending := []string{
    `V:\`
}
```

Acts as a queue of directories waiting to be scanned.

Example:

```text
[V:\Movies]
```

Worker discovers:

```text
Action
Comedy
Drama
```

Dispatcher appends:

```text
[
    V:\Movies\Action,
    V:\Movies\Comedy,
    V:\Movies\Drama,
]
```

This creates a breadth-first traversal of the filesystem.

---

## Results Channel

Workers return:

```go
type Results struct {
    newDirs []string
}
```

Example:

```go
Results{
    newDirs: []string{
        "V:\\Movies\\Action",
        "V:\\Movies\\Comedy",
    },
}
```

Dispatcher receives the result and adds those directories to the pending queue.

---

## Active Counter

Tracks currently running directory scans.

When dispatching work:

```go
active++
```

When a worker finishes:

```go
active--
```

Traversal is complete when:

```go
active == 0 &&
len(pending) == 0
```

Meaning:

* No workers are busy
* No directories remain to be scanned

---

## File Counter

```go
var fileCounter atomic.Int64
```

Counts discovered `.mp4` files.

Workers increment:

```go
fileCounter.Add(1)
```

Atomics are used because multiple workers update the counter concurrently.

---

## SafeMap

```go
type SafeMap struct {
    mu sync.Mutex
    sm map[int64][]string
}
```

Stores:

```text
File Size -> List of File Paths
```

Example:

```go
map[
    1048576: [
        movie1.mp4,
        movie2.mp4,
    ]
]
```

Mutex protects the map from concurrent writes.

Without the mutex:

```go
fatal error:
concurrent map writes
```

would eventually occur.

---

## Grouping By File Size

For every mp4:

```go
info, _ := entry.Info()
size := info.Size()
```

Store:

```go
sizeMap.sm[size] = append(
    sizeMap.sm[size],
    fullpath,
)
```

Result:

```text
1048576 bytes
    movie1.mp4
    movie2.mp4

2097152 bytes
    movie3.mp4
```

---

## Why Group By Size?

Example:

```text
movie1.mp4   1.5 GB
movie2.mp4   1.5 GB
movie3.mp4   2.0 GB
```

Only files with matching sizes can possibly be duplicates.

After grouping:

```text
1.5 GB
    movie1.mp4
    movie2.mp4
```

These become candidates for hashing.

---

## Next Step

Current algorithm:

```text
Find MP4 files
    ↓
Group by file size
```

Future duplicate finder:

```text
Find MP4 files
    ↓
Group by size
    ↓
Only keep groups with >1 file
    ↓
Calculate SHA256 / MD5
    ↓
Group by hash
    ↓
Actual duplicates
```

This avoids hashing every file on the drive.

---

## Pattern Used

Concurrency Pattern:

```text
Single Dispatcher + Worker Pool
```

Traversal Strategy:

```text
Breadth First Search (BFS)
```

Data Aggregation:

```text
Grouping using map[int64][]string
```

Thread Safety:

```text
sync.Mutex
atomic.Int64
```
