# JSON files for persistence

We store each Session as one JSON file in a data directory (a Docker volume). Writes are atomic (temp file + rename) and serialized behind a mutex. The stack is stdlib-only Go in a `FROM scratch` image: SQLite would need cgo or a third-party driver, and a separate database server is overkill for the very low expected traffic. Sessions are never deleted.
