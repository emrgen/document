# document is a document store for small projects

document is a document store for small projects. It is designed to be simple and easy to use.

## Features

- [x] Create document
- [x] Update document
- [x] Delete document
- [x] Get document
- [x] Get document by ids(full docs)
- [x] List documents
- [ ] Search documents
- [x] Document versioning
- [x] Document history
- [ ] Document tags
- [x] Document backup
- [ ] Document restore
- [ ] Document export
- [x] Document backlinks
- [x] Document links
- [ ] Document auto backup to S3
- [ ] Document auto load from S3
- [x] Create a job to clean up old documents backups, (keep backups at 10min interval)

## Installation

```bash
# install initial dependencies(it will fail but that's fine, still need to run it)
make deps
# build proto
make protoc
# install all dependencies
make deps
```

## Usage
