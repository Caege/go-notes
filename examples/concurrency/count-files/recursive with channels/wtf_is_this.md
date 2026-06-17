# How this recursive directory crawler works

## High-level idea

The program walks a directory tree concurrently.

Each `processFolder()` goroutine:

1. Reads one directory.
2. Sends discovered subdirectories to `folderCh`.
3. Sends discovered files to `fileCh`.
4. Marks itself done via `wg.Done()`.

The main goroutine acts as a coordinator:

1. Receives directories from `folderCh`.
2. Starts a new `processFolder()` goroutine for each directory.
3. Receives files from `fileCh` and counts them.
4. Exits when both channels are closed.

---

## Work flow

Initial state:

```text
main
 └── processFolder(C:\)
```

When `processFolder(C:\)` finds:

```text
C:\
 ├── Users
 ├── Program Files
 └── file.txt
```

it does:

```go
wg.Add(1)
folderCh <- "C:\\Users"

wg.Add(1)
folderCh <- "C:\\Program Files"

fileCh <- file.txt
```

Main receives those folders and starts new goroutines:

```text
main
 ├── processFolder(C:\Users)
 └── processFolder(C:\Program Files)
```

Each discovered directory becomes a new unit of work.

---

## Why WaitGroup is used

The WaitGroup tracks the number of directories that still need processing.

Before a new directory is scheduled:

```go
wg.Add(1)
folderCh <- childDir
```

When a directory finishes:

```go
defer wg.Done()
```

Conceptually:

```text
Add(1)  => "one more directory exists"
Done()  => "that directory is finished"
```

When the count reaches zero:

```go
wg.Wait()
```

returns.

That means no directory processing goroutines remain.

---

## Why Add happens before sending to folderCh

This is important.

Correct:

```go
wg.Add(1)
folderCh <- childDir
```

Incorrect:

```go
folderCh <- childDir
wg.Add(1)
```

The count must increase before the work is made visible.

Otherwise a worker could finish and cause the WaitGroup count to reach zero before the new directory has been accounted for.

Think of `Add(1)` as reserving a slot for future work.

---

## Why main launches new goroutines without calling Add

Main does:

```go
case folder := <-folderCh:
    go processFolder(folder, ...)
```

Notice there is no:

```go
wg.Add(1)
```

here.

That's because the sender already performed:

```go
wg.Add(1)
folderCh <- folder
```

The WaitGroup count belongs to the directory itself, not to the goroutine creation.

By the time main receives the folder, its work has already been counted.

---

## Purpose of the channel closer goroutine

```go
go func() {
    wg.Wait()

    close(fileCh)
    close(folderCh)
}()
```

This goroutine waits until all directory work is finished.

Once the WaitGroup reaches zero:

```text
No directories left to process
```

Therefore no more values will ever be sent to either channel.

At that point it is safe to close them.

---

## Why main sets channels to nil

When a channel closes:

```go
v, ok := <-ch
```

returns:

```go
ok == false
```

Then main does:

```go
fileCh = nil
```

or

```go
folderCh = nil
```

A nil channel is ignored by `select`.

This removes that case from future iterations.

Without this, the closed channel would always be ready and the loop would spin forever.

---

## Loop termination

The loop condition:

```go
for folderCh != nil || fileCh != nil
```

means:

```text
Keep running while at least one channel is still active.
```

Eventually:

```text
folderCh = nil
fileCh = nil
```

and the loop exits.

---

## Important caveat

This program appears to work, but it relies on a subtle WaitGroup pattern:

```go
wg.Add(1)
```

can happen while another goroutine is already blocked in:

```go
wg.Wait()
```

The Go documentation discourages this pattern because it can lead to races in more complex designs.

For learning purposes this demonstrates recursive work discovery nicely, but for production code a worker-pool or explicit work queue is usually safer and easier to reason about.

---

## Mental model

Think of the WaitGroup count as:

```text
"Number of directories that still exist to be processed"
```

Every discovered directory:

```text
Add(1)
```

Every finished directory:

```text
Done()
```

When the count reaches zero:

```text
No directories remain.
Close channels.
Exit.
```

That is the core idea behind the entire program.
