# document is a document store for small projects

document is a document store for small projects. It is designed to be simple and easy to use.

## Design

All documents are treated as a single entity.
Document can add links to other documents. Document can add children documents.
Children documents are treated as a single independent document. They can have their own links and children.
Each document is published with a version number. Each update to the document will increase the version number.
One document can have multiple parents. When a document is published, optionally it can publish all children documents as well.
This will create a new version of the document and all children documents. Other parent documents will stay linked to the @current version of the document.


## Progress

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
- [x] Document restore
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
