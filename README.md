# weaveTest

A Go-based project utilizing gRPC.

## Overview

weaveTest is a Go application built with the latest Go 1.24.1 runtime. This project follows standard Go project layout
with separate cmd and internal directories.

## Requirements

- Go 1.24.1 or higher
- Dependencies as listed in go.mod

## Installation
Copy the .env.example.json into .env.json. It should look something like this
```
{
  "git_token": "your pat",
  "pagination": {
    "rate_limited": false,
    "kind": "single",
    "fetch_all_pages": false,
    "max_pages": 5,
    "per_page": 10
  }
}
```
To run and install all deps ensure go has been installed and run the following command.
```aiignore
make run
```
It runs on port 50051.
Features that have been implemented
* SinglePagePaginator
* MultiPagePaginator
Ongoing
* RateLimiter(sliding window)
* tests (tests for query string exist)
