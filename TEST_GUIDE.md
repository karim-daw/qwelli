# Testing Qwelli CLI

## Prerequisites

You need an OpenAI API key. If you don't have one:
1. Go to https://platform.openai.com/api-keys
2. Create a new API key
3. Copy it (you'll need it in step 2 below)

## Step-by-Step Test

### 1. Build the Binary

```bash
cd /home/karim-daw/dev/qwelli
go build -o qwelli ./cmd/qwelli
```

### 2. Initialize Configuration

```bash
./qwelli init
```

**You'll be prompted for:**
- OpenAI API key: `sk-...` (paste your key)
- Embedding model: Just press Enter to use default (`text-embedding-3-small`)
- API endpoint: Just press Enter to use default

**Expected output:**
```
🚀 Welcome to Qwelli!
Let's set up your configuration.

Enter your OpenAI API key: sk-...
Enter embedding model [text-embedding-3-small]: 
Enter API endpoint [https://api.openai.com/v1/embeddings]: 

✅ Configuration saved!
📁 Config file: /home/karim-daw/.qwelli/config.yaml
📁 Index directory: /home/karim-daw/.qwelli/indexes

You can now run 'qwelli index <folder>' to start indexing!
```

### 3. Index the Test Data

```bash
./qwelli index tests/demo/testdata
```

**Expected output:**
```
📂 Indexing folder: /home/karim-daw/dev/qwelli/tests/demo/testdata
💾 Database: /home/karim-daw/.qwelli/indexes/testdata.db

📄 Processing 13/13: technical_doc.txt
✅ Indexing completed in 3.5s
🔍 You can now search with: qwelli search "your query" --index tests/demo/testdata
```

### 4. Search the Indexed Files

```bash
# Search for greetings
./qwelli search "hello greeting" --index tests/demo/testdata

# Search for farewells
./qwelli search "goodbye farewell" --index tests/demo/testdata

# Search for technical content
./qwelli search "machine learning algorithms" --index tests/demo/testdata

# Search for recipes
./qwelli search "chocolate chip cookies" --index tests/demo/testdata

# Get more results
./qwelli search "python code" --index tests/demo/testdata --top 10
```

**Expected output:**
```
🔍 Searching for: "hello greeting"

Result 1:
  📄 File: hello.txt
  📁 Path: /home/karim-daw/dev/qwelli/tests/demo/testdata/hello.txt
  📏 Distance: 0.2345
  📝 Preview: Hello! This is a greeting file...

Result 2:
  📄 File: meeting_notes.md
  📁 Path: /home/karim-daw/dev/qwelli/tests/demo/testdata/meeting_notes.md
  📏 Distance: 0.4567
  📝 Preview: Meeting started with greetings...
```

### 5. List All Indexes

```bash
./qwelli list
```

**Expected output:**
```
📚 Indexed folders (1):

1. testdata
   📄 Documents: 13
   💾 Database: /home/karim-daw/.qwelli/indexes/testdata.db
   🕐 Last modified: 2025-12-07 21:30:45
```

### 6. Check Index Status

```bash
./qwelli status --index tests/demo/testdata
```

**Expected output:**
```
📊 Index Status: /home/karim-daw/dev/qwelli/tests/demo/testdata

📄 Indexed documents: 13
💾 Database: /home/karim-daw/.qwelli/indexes/testdata.db
💽 Database size: 2.34 MB
🕐 Last indexed: 2025-12-07 21:30:45
```

### 7. Index Your Own Folder

```bash
# Index any folder you want
./qwelli index ~/Documents/my-project

# Then search it
./qwelli search "whatever you're looking for" --index ~/Documents/my-project
```

## Quick Test Script

Here's a one-shot test script:

```bash
#!/bin/bash

# Build
go build -o qwelli ./cmd/qwelli

# Note: You need to run init manually first to provide API key
# ./qwelli init

# Index test data
./qwelli index tests/demo/testdata

# Run some searches
echo "Testing searches..."
./qwelli search "hello greeting" --index tests/demo/testdata
./qwelli search "python programming" --index tests/demo/testdata
./qwelli search "recipe cooking" --index tests/demo/testdata

# List and status
./qwelli list
./qwelli status --index tests/demo/testdata

echo "✅ All tests completed!"
```

## Troubleshooting

### "config not found" error
```
Run 'qwelli init' first
```

### "index not found" error
```
Run 'qwelli index <folder>' first for that folder
```

### OpenAI API error
```
Check your API key in ~/.qwelli/config.yaml
Make sure you have credits in your OpenAI account
```

### Permission denied
```bash
chmod +x ./qwelli
```

## Verify Installation

Check where your data is stored:

```bash
# Config file
cat ~/.qwelli/config.yaml

# Index databases
ls -lh ~/.qwelli/indexes/

# Check database size
du -sh ~/.qwelli/indexes/*.db
```

## Clean Up Test Data

```bash
# Remove test index
rm ~/.qwelli/indexes/testdata.db

# Remove all data
rm -rf ~/.qwelli/
```
