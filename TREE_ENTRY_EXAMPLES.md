# Git Tree Entry Examples

This document provides concrete examples of how Git tree entries work.

## Example 1: Simple Repository

Let's say you have this file structure:

```
my-project/
├── README.md
├── app.js
└── package.json
```

### Step 1: Creating Blob Objects

When you stage these files, Git creates blob objects for each file's content:

```bash
# Content of README.md: "# My Project"
blob 1a2b3c4d → "# My Project"

# Content of app.js: "console.log('Hello');"
blob 5e6f7g8h → "console.log('Hello');"

# Content of package.json: '{"name": "my-app"}'
blob 9i0j1k2l → '{"name": "my-app"}'
```

### Step 2: Creating Tree Entries

The root tree object contains three entries:

```
Tree Object (SHA: abc123def456)
├── TreeEntry 1:
│   ├── mode: 100644 (regular file)
│   ├── name: "README.md"
│   └── sha:  1a2b3c4d (points to blob)
│
├── TreeEntry 2:
│   ├── mode: 100644 (regular file)
│   ├── name: "app.js"
│   └── sha:  5e6f7g8h (points to blob)
│
└── TreeEntry 3:
    ├── mode: 100644 (regular file)
    ├── name: "package.json"
    └── sha:  9i0j1k2l (points to blob)
```

### Binary Representation

Each entry is serialized as:

```
100644 README.md\0[20 bytes: 1a2b3c4d...]
100644 app.js\0[20 bytes: 5e6f7g8h...]
100644 package.json\0[20 bytes: 9i0j1k2l...]
```

---

## Example 2: Nested Directories

Now with subdirectories:

```
my-project/
├── README.md
├── src/
│   ├── main.go
│   └── utils/
│       └── helper.go
└── tests/
    └── main_test.go
```

### Step 1: Create Blobs

```
blob aa11bb22 → content of README.md
blob cc33dd44 → content of main.go
blob ee55ff66 → content of helper.go
blob gg77hh88 → content of main_test.go
```

### Step 2: Build Tree Hierarchy (Bottom-Up)

#### Deepest Level: utils/ tree

```
Tree Object (SHA: tree111)
└── TreeEntry:
    ├── mode: 100644
    ├── name: "helper.go"
    └── sha:  ee55ff66 (blob)
```

#### Middle Level: src/ tree

```
Tree Object (SHA: tree222)
├── TreeEntry 1:
│   ├── mode: 100644
│   ├── name: "main.go"
│   └── sha:  cc33dd44 (blob)
│
└── TreeEntry 2:
    ├── mode: 040000 (directory!)
    ├── name: "utils"
    └── sha:  tree111 (points to utils/ tree)
```

#### Middle Level: tests/ tree

```
Tree Object (SHA: tree333)
└── TreeEntry:
    ├── mode: 100644
    ├── name: "main_test.go"
    └── sha:  gg77hh88 (blob)
```

#### Top Level: Root tree

```
Tree Object (SHA: root999)
├── TreeEntry 1:
│   ├── mode: 100644
│   ├── name: "README.md"
│   └── sha:  aa11bb22 (blob)
│
├── TreeEntry 2:
│   ├── mode: 040000 (directory!)
│   ├── name: "src"
│   └── sha:  tree222 (points to src/ tree)
│
└── TreeEntry 3:
    ├── mode: 040000 (directory!)
    ├── name: "tests"
    └── sha:  tree333 (points to tests/ tree)
```

### Visual Flow

```
root999 (root tree)
  |
  ├─→ README.md (aa11bb22 blob)
  ├─→ src (tree222)
  |     ├─→ main.go (cc33dd44 blob)
  |     └─→ utils (tree111)
  |           └─→ helper.go (ee55ff66 blob)
  └─→ tests (tree333)
        └─→ main_test.go (gg77hh88 blob)
```

---

## Example 3: Executable Files and Symlinks

```
project/
├── script.sh        (executable)
├── config.yaml      (regular file)
└── link.txt         (symlink to config.yaml)
```

### Tree Entries with Different Modes

```
Tree Object (SHA: xyz789)
├── TreeEntry 1:
│   ├── mode: 100644 (regular file)
│   ├── name: "config.yaml"
│   └── sha:  blob123
│
├── TreeEntry 2:
│   ├── mode: 120000 (symlink)
│   ├── name: "link.txt"
│   └── sha:  blob456 (blob contains "config.yaml")
│
└── TreeEntry 3:
    ├── mode: 100755 (executable!)
    ├── name: "script.sh"
    └── sha:  blob789
```

---

## Example 4: How Git Reuses Trees

Let's say you modify only one file:

### Initial Commit

```
Commit A
└─→ root-tree-v1
    ├─→ README.md (blob-aaa)
    └─→ src/ (tree-src-v1)
        ├─→ main.go (blob-bbb)
        └─→ util.go (blob-ccc)
```

### After Modifying main.go

```
Commit B
└─→ root-tree-v2 (NEW)
    ├─→ README.md (blob-aaa) ← REUSED (same blob)
    └─→ src/ (tree-src-v2) (NEW)
        ├─→ main.go (blob-ddd) ← CHANGED (new blob)
        └─→ util.go (blob-ccc) ← REUSED (same blob)
```

**Key Points:**

- Only `main.go` changed, so only its blob is new (`blob-ddd`)
- `util.go` and `README.md` blobs are reused
- But Git creates new tree objects for `src/` and root because they contain different entries

---

## Example 5: Real Git Commands

Let's trace actual Git internals:

```bash
# Create a simple repo
mkdir demo && cd demo
git init

# Add files
echo "Hello" > file.txt
mkdir src
echo "Code" > src/main.js

# Stage files
git add .

# Look at what's staged
git ls-files --stage
# Output:
# 100644 e965047ad7c57865823c7d992b1d046ea66edf78 0	file.txt
# 100644 2b546e8d4f1b3c5f8e09a17b94f6e2c9d4a3f1e0 0	src/main.js

# These are the blob SHAs!
```

```bash
# After commit, view the tree
git cat-file -p HEAD^{tree}
# Output:
# 100644 blob e965047ad7c57865823c7d992b1d046ea66edf78    file.txt
# 040000 tree a1b2c3d4e5f6789012345678901234567890abcd    src

# View the src/ tree
git cat-file -p a1b2c3d4e5f6789012345678901234567890abcd
# Output:
# 100644 blob 2b546e8d4f1b3c5f8e09a17b94f6e2c9d4a3f1e0    main.js
```

---

## Example 6: TreeEntry Validation

The code validates tree entries. Here's what's allowed and what's not:

### ✅ Valid Tree Entries

```go
NewTreeEntry(FileModeRegular, "hello.txt", "abc123...")      // OK
NewTreeEntry(FileModeDirectory, "src", "def456...")          // OK
NewTreeEntry(FileModeExecutable, "build.sh", "ghi789...")    // OK
NewTreeEntry(FileModeRegular, "my-file.name", "jkl012...")   // OK (hyphens, dots)
```

### ❌ Invalid Tree Entries

```go
NewTreeEntry(FileModeRegular, "", "abc123...")               // ERROR: empty name
NewTreeEntry(FileModeRegular, ".", "abc123...")              // ERROR: invalid name
NewTreeEntry(FileModeRegular, "src/main.go", "abc123...")    // ERROR: contains '/'
NewTreeEntry(FileModeRegular, "dir\\file", "abc123...")      // ERROR: contains '\'
```

**Why?** Tree entries represent **single-level directory items** only. Paths are built by nesting trees, not by including slashes in names.

---

## Example 7: Sorting Order

Git sorts tree entries in a specific way:

```
Tree entries are sorted as if directories have a trailing '/'
```

Given these files:

```
file
file.txt
file2
src/
src-old/
```

Git's sorting (for tree entries):

```
1. file          (100644 file)
2. file.txt      (100644 file.txt)
3. file2         (100644 file2)
4. src/          (040000 src)      ← treated as "src/"
5. src-old/      (040000 src-old)  ← treated as "src-old/"
```

The `CompareTo` method ensures directories come before files with similar names.

---

## Example 8: Practical Usage in Your Code

Here's how `tree_builder.go` uses tree entries:

```go
// 1. You have index entries (staged files)
idx.Entries = [
    {Path: "README.md", BlobHash: "abc123"},
    {Path: "src/main.go", BlobHash: "def456"},
]

// 2. TreeBuilder builds nested structure
root := directoryNode{
    files: {"README.md": "abc123"},
    subdirs: {
        "src": {
            files: {"main.go": "def456"}
        }
    }
}

// 3. For src/ directory, create tree entry
srcTreeEntry := NewTreeEntry(
    FileModeDirectory,     // mode: 040000
    "src",                 // name: just "src", not "src/main.go"
    "tree789"              // SHA of the src/ tree object
)

// 4. For README.md, create tree entry
readmeEntry := NewTreeEntry(
    FileModeRegular,       // mode: 100644
    "README.md",           // name
    "abc123"               // blob SHA
)

// 5. Root tree contains both entries
rootTree := NewTree([]*TreeEntry{
    readmeEntry,
    srcTreeEntry,
})
```

---

## Key Takeaways

1. **Tree entries are pointers** - They don't contain file content, just references (SHA-1 hashes)

2. **Hierarchies through nesting** - Deep paths are built by linking tree objects, not by putting paths in entry names

3. **Content-addressed** - Same content = same SHA = reused object

4. **Immutable** - Once created, tree objects never change

5. **Efficient storage** - Git only stores what changed; unchanged files/trees are reused across commits
