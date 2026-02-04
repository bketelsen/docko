# docko

Docko is an application for managing a corpus of PDF files. It should provide a graphical interface to search for existing documents by contents (full text search), tag (user-applied or auto-generated), or correspondent (the person or company that created the document)

## Document Store

Docku is primarily a document store meant to be the final system of record for PDFs.

The document store will consist of a directory tree with folders for:

- inbox: unprocessed documents
- originals: copy of unaltered original PDFs
- documents: the main store of PDF files, possibly altered from original format
- thumbnails: generated thumbnail images

The document store will also have a document metadata database that keeps relevant information about document, tags, and correspondents:

- original document location: see Document Sources
- original name
- file size
- page count
- tags
- correspondents
- etc.

## Document Sources

Docku will have a local "inbox" directory it will watch for unprocessed PDFs.

Docku can optionally use Go libraries for SMB and NFS connections configured in the admin interface. This allows the user to specify a network share as an additional source of documents to import into the system. Docku will not require the user to create actual filesystem mounts to use document sources, it will only use SMB/NFS libraries to copy/move/delete PDF files.

## Batch Management System

All tasks related to document processing will be handled by a queue system. The Dashboard will provide visibility into the queue showing counts of completed and uncompleted tasks for each type of queue. The queue management system will keep a detailed audit record of the processing and movement of each file through the system. It should be easy for a user to see all the queue history for a document, each step that happened along the processing pipeline should be logged and tracked.

Each queue should be configurable for the number of concurrent processors.

## Processing Documents

When a new document is found in one of the Document Sources, it should be added into an initial queue for processing.

Documents should be assigned a UUID upon entering the system. The document should be named {UUID}.pdf in the file system. Metadata stored in the database will keep the document's actual title.

Document processing should perform the following tasks:

- duplicate checking: does this document match a document that already exists in docku?
- full text indexing
- automatic tagging: use the OpenAI SDK to submit the document with a prompt to get the most relevant tags for the document. Allow limiting the amount of text submitted so that huge PDFs like books aren't sent in their entirety, just the first N pages. Make the limit user-configurable.
- Correspondents: Attempt to discern the entity that sent the document. For example a PDF that is a utility bill may have "Duke Energy" as the source. Create robust correspondent detection and deduplication so that there aren't correspondents named "Duke", "Duke Energy", "Duke Energy, Inc".

Each of these processing steps should be its own queue.

## Duplicate Document Handling

Each Document Source should be configurable so that if a document is processed from that source and determined to be a duplicate of a document already existing in Docku, the system will either delete the duplicate or rename it with some pre-configured pattern ("my-document.duplicate-of-XXXX.pdf)

## New

- min number of words import setting to prevent importing image-only pdfs
- dashboard at / with actual stats
- refactor to use all/more the templui components
- expander/accordian with details on /queues
- sidebar: new admin section with inboxes, sources, queues
- constants: instead of "magic strings" (like "ai_complete", or "processed") - replace with go const strings
- deep integration tests: network sources, delete/rename testing, duplicates. Ensure that if configured properly, I end up with one copy of a PDF and others are clearly marked dups or deleted.
