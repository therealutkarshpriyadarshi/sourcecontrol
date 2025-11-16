# Worker Pool Pattern - Production Guide

This guide explains the production-ready worker pool implementation used in this package.

## Overview

The worker pool pattern is a concurrency design pattern that manages a fixed number of workers to process tasks concurrently. This implementation provides:

- **Type Safety**: Generic implementation works with any task and result types
- **Context Support**: Full support for cancellation and timeouts
- **Error Handling**: Proper error propagation with fail-fast behavior
- **Panic Recovery**: Workers recover from panics and convert them to errors
- **Resource Management**: Automatic cleanup of goroutines and channels
- **Production Ready**: Battle-tested patterns used in production systems

## Key Features

### 1. Generic Types
The worker pool uses Go generics to work with any task and result types:

```go
pool := NewWorkerPool[InputType, OutputType]()
```

### 2. Configurable Workers
Control the number of concurrent workers and buffer sizes:

```go
// Use default (runtime.GOMAXPROCS)
pool := NewWorkerPool[int, int]()

// Custom worker count
pool := NewWorkerPool[int, int](
    WithWorkerCount(8),
    WithTaskBuffer(16),
)
```

### 3. Context-Aware
Full support for context cancellation and timeouts:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

results, err := pool.Process(ctx, tasks, processFn)
```

### 4. Error Handling
Fail-fast behavior: if any worker returns an error, all workers are cancelled:

```go
processFn := func(ctx context.Context, task Task) (Result, error) {
    if task.invalid {
        return nil, fmt.Errorf("invalid task")
    }
    return processTask(task), nil
}

results, err := pool.Process(ctx, tasks, processFn)
if err != nil {
    // Handle error - partial results may be available
    log.Printf("Processing failed: %v", err)
}
```

## API Methods

### 1. Process - Slice-based Processing

Process a slice of tasks and return results in the same order:

```go
func (wp *WorkerPool[T, R]) Process(
    ctx context.Context,
    tasks []T,
    processFn ProcessFunc[T, R],
) ([]R, error)
```

**Use Case**: When you have a known list of tasks and need results in order.

**Example**:
```go
pool := NewWorkerPool[string, int]()

tasks := []string{"file1.txt", "file2.txt", "file3.txt"}
processFn := func(ctx context.Context, path string) (int, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return 0, err
    }
    return len(data), nil
}

sizes, err := pool.Process(context.Background(), tasks, processFn)
// sizes[0] = size of file1.txt
// sizes[1] = size of file2.txt
// sizes[2] = size of file3.txt
```

### 2. ProcessMap - Map-based Processing

Process a map of tasks and return a map of results:

```go
func (wp *WorkerPool[T, R]) ProcessMap(
    ctx context.Context,
    tasks map[string]T,
    processFn ProcessFunc[T, R],
) (map[string]R, error)
```

**Use Case**: When tasks are naturally represented as key-value pairs.

**Example**:
```go
pool := NewWorkerPool[FileInfo, []byte]()

files := map[string]FileInfo{
    "config.json": configInfo,
    "data.json":   dataInfo,
}

processFn := func(ctx context.Context, info FileInfo) ([]byte, error) {
    return readFileContent(info)
}

results, err := pool.ProcessMap(context.Background(), files, processFn)
// results["config.json"] = config file contents
// results["data.json"] = data file contents
```

### 3. ProcessStream - Channel-based Processing

Process tasks from an input channel and stream results:

```go
func (wp *WorkerPool[T, R]) ProcessStream(
    ctx context.Context,
    taskChan <-chan T,
    processFn ProcessFunc[T, R],
) (<-chan R, <-chan error)
```

**Use Case**: For streaming scenarios where tasks arrive dynamically.

**Example**:
```go
pool := NewWorkerPool[Request, Response]()

taskChan := make(chan Request, 100)
go func() {
    defer close(taskChan)
    for req := range incomingRequests {
        taskChan <- req
    }
}()

processFn := func(ctx context.Context, req Request) (Response, error) {
    return handleRequest(req)
}

resultChan, errChan := pool.ProcessStream(context.Background(), taskChan, processFn)

// Process results as they arrive
for result := range resultChan {
    handleResult(result)
}

// Check for errors
if err := <-errChan; err != nil {
    log.Printf("Error during processing: %v", err)
}
```

## Real-World Examples

### Example 1: Tree File Processing (From analyzer.go)

```go
// Process multiple directories concurrently
type dirTask struct {
    sha  objects.ObjectHash
    path scpath.RelativePath
}

pool := NewWorkerPool[dirTask, FileMap]()

processFn := func(ctx context.Context, task dirTask) (FileMap, error) {
    return a.getTreeFiles(task.sha, task.path)
}

results, err := pool.Process(context.Background(), directories, processFn)
if err != nil {
    return nil, err
}

// Merge all results
for _, subFiles := range results {
    maps.Copy(files, subFiles)
}
```

**Why this works**:
- Each directory can be processed independently
- Results maintain order for predictable merging
- Context support allows cancellation if one directory fails

### Example 2: Index Entry Creation (From indexer.go)

```go
// Create index entries for multiple files concurrently
type task struct {
    path scpath.RelativePath
    info FileInfo
}

pool := NewWorkerPool[task, *index.Entry]()

// Convert map to slice of tasks
tasks := make([]task, 0, len(targetFiles))
for path, info := range targetFiles {
    tasks = append(tasks, task{path: path, info: info})
}

processFn := func(ctx context.Context, t task) (*index.Entry, error) {
    entry, err := u.createIndexEntry(t.path, t.info)
    if err != nil {
        return nil, fmt.Errorf("create entry for %s: %w", t.path, err)
    }
    return entry, nil
}

entries, err := pool.Process(context.Background(), tasks, processFn)
```

**Why this works**:
- I/O-bound operations (file stats) benefit from concurrency
- Error handling ensures no partial/corrupt index
- Automatic retry/cancellation on failure

## Best Practices

### 1. Choose the Right Worker Count

```go
// CPU-bound tasks: use GOMAXPROCS (default)
pool := NewWorkerPool[Task, Result]()

// I/O-bound tasks: can use more workers
pool := NewWorkerPool[Task, Result](
    WithWorkerCount(runtime.GOMAXPROCS(0) * 4),
)

// Rate-limited APIs: use conservative count
pool := NewWorkerPool[APIRequest, APIResponse](
    WithWorkerCount(10), // Respect API rate limits
)
```

### 2. Set Appropriate Timeouts

```go
// Overall timeout for all tasks
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

results, err := pool.Process(ctx, tasks, processFn)
```

### 3. Handle Partial Results on Error

```go
results, err := pool.Process(ctx, tasks, processFn)
if err != nil {
    // Some results may still be valid
    for i, result := range results {
        if isValid(result) {
            useResult(result)
        }
    }
    return fmt.Errorf("partial failure: %w", err)
}
```

### 4. Don't Create New Pools Unnecessarily

```go
// BAD: Creating a new pool for each call
func processFiles(files []string) error {
    pool := NewWorkerPool[string, int]()  // Creates goroutines
    return pool.Process(ctx, files, processFn)
}

// GOOD: Reuse pool or pass as parameter
type Service struct {
    pool *WorkerPool[string, int]
}

func NewService() *Service {
    return &Service{
        pool: NewWorkerPool[string, int](),
    }
}

func (s *Service) processFiles(files []string) error {
    return s.pool.Process(ctx, files, processFn)
}
```

### 5. Use Appropriate Buffer Sizes

```go
// Small tasks, fast processing: small buffer
pool := NewWorkerPool[int, int](
    WithTaskBuffer(10),
)

// Large tasks, slow processing: larger buffer
pool := NewWorkerPool[LargeTask, Result](
    WithTaskBuffer(100),
)
```

## Performance Characteristics

### Time Complexity
- **Setup**: O(W) where W = number of workers
- **Processing**: O(N/W) where N = number of tasks
- **Cleanup**: O(W)

### Space Complexity
- **Memory**: O(W + B) where B = buffer size
- **Goroutines**: Exactly W workers + 2 management goroutines

### When to Use
✅ **Good for**:
- I/O-bound tasks (file operations, network calls)
- CPU-bound tasks with many iterations
- Tasks that can be processed independently
- Batch processing with predictable load

❌ **Avoid for**:
- Extremely fast tasks (< 1μs) - overhead dominates
- Tasks with complex dependencies
- Real-time processing with strict latency requirements
- Single-digit task counts

## Comparison with Alternatives

### vs. Manual Goroutines + WaitGroup

**Manual Approach**:
```go
var wg sync.WaitGroup
results := make([]Result, len(tasks))
for i, task := range tasks {
    wg.Add(1)
    go func(i int, t Task) {
        defer wg.Done()
        results[i] = process(t)
    }(i, task)
}
wg.Wait()
```

**Problems**:
- No worker limit (can spawn thousands of goroutines)
- No context support
- No error handling
- Manual result collection

**Worker Pool Approach**:
```go
results, err := pool.Process(ctx, tasks, processFn)
```

**Benefits**:
- Controlled concurrency
- Context cancellation
- Automatic error propagation
- Clean API

### vs. errgroup.Group

**errgroup Approach**:
```go
g, ctx := errgroup.WithContext(ctx)
for _, task := range tasks {
    task := task
    g.Go(func() error {
        return process(ctx, task)
    })
}
err := g.Wait()
```

**When to use errgroup**: Simple parallel operations without result collection

**When to use WorkerPool**:
- Need to collect and order results
- Want to limit concurrent workers
- Processing hundreds/thousands of tasks
- Need map-based or streaming processing

### vs. golang.org/x/sync/semaphore

**Semaphore** is great for limiting concurrency but:
- No result collection
- More boilerplate
- No streaming support

**WorkerPool** provides higher-level abstractions with result handling.

## Testing Strategies

### 1. Test with Small Task Counts
```go
func TestProcess_SmallTaskCount(t *testing.T) {
    pool := NewWorkerPool[int, int](WithWorkerCount(4))
    tasks := []int{1, 2, 3}
    results, err := pool.Process(ctx, tasks, processFn)
    // Verify all tasks processed correctly
}
```

### 2. Test Error Handling
```go
func TestProcess_ErrorPropagation(t *testing.T) {
    pool := NewWorkerPool[int, int]()
    tasks := []int{1, 2, 3, 4, 5}
    processFn := func(ctx context.Context, task int) (int, error) {
        if task == 3 {
            return 0, errors.New("failed")
        }
        return task, nil
    }
    _, err := pool.Process(ctx, tasks, processFn)
    if err == nil {
        t.Fatal("expected error")
    }
}
```

### 3. Test Context Cancellation
```go
func TestProcess_Cancellation(t *testing.T) {
    pool := NewWorkerPool[int, int]()
    ctx, cancel := context.WithCancel(context.Background())

    processFn := func(ctx context.Context, task int) (int, error) {
        time.Sleep(100 * time.Millisecond)
        return task, nil
    }

    go func() {
        time.Sleep(50 * time.Millisecond)
        cancel()
    }()

    _, err := pool.Process(ctx, tasks, processFn)
    if !errors.Is(err, context.Canceled) {
        t.Error("expected context.Canceled")
    }
}
```

## Troubleshooting

### Problem: Deadlock
**Symptom**: Program hangs indefinitely

**Common Causes**:
1. Not closing input channels in ProcessStream
2. Workers blocked on channel operations
3. Context not properly propagated

**Solution**: Always close task channels and check context

### Problem: Memory Growth
**Symptom**: Increasing memory usage during processing

**Common Causes**:
1. Buffer size too large
2. Results not consumed fast enough (ProcessStream)
3. Large result objects retained

**Solution**:
- Use smaller buffers
- Process results as they arrive
- Profile with `pprof`

### Problem: Poor Performance
**Symptom**: Worker pool slower than sequential

**Common Causes**:
1. Tasks too fast (< 1μs each)
2. Too many workers for CPU-bound tasks
3. Excessive synchronization

**Solution**:
- Batch small tasks together
- Tune worker count based on task type
- Benchmark different configurations

## Migration Guide

### From Manual Goroutines

**Before**:
```go
results := make(chan result, len(tasks))
var wg sync.WaitGroup

for _, task := range tasks {
    wg.Add(1)
    go func(t Task) {
        defer wg.Done()
        results <- process(t)
    }(task)
}

wg.Wait()
close(results)
```

**After**:
```go
pool := NewWorkerPool[Task, Result]()
results, err := pool.Process(ctx, tasks, processFn)
```

### From errgroup.Group (with result collection)

**Before**:
```go
g, ctx := errgroup.WithContext(ctx)
results := make([]Result, len(tasks))
var mu sync.Mutex

for i, task := range tasks {
    i, task := i, task
    g.Go(func() error {
        result, err := process(ctx, task)
        if err != nil {
            return err
        }
        mu.Lock()
        results[i] = result
        mu.Unlock()
        return nil
    })
}

err := g.Wait()
```

**After**:
```go
pool := NewWorkerPool[Task, Result]()
results, err := pool.Process(ctx, tasks, processFn)
```

## Conclusion

The worker pool pattern provides a production-ready solution for concurrent task processing in Go. Key benefits:

1. **Type Safety**: Generic implementation prevents type errors
2. **Resource Control**: Fixed number of workers prevents resource exhaustion
3. **Error Handling**: Proper error propagation and panic recovery
4. **Context Support**: Standard cancellation and timeout patterns
5. **Clean API**: Simple, intuitive interface

Use this pattern when you need controlled concurrency with proper error handling and result collection.
