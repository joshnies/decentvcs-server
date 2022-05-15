# Quanta Control API

## Requirements
|Dependency|Version|
|-|-|
|go|1.18+|

## Setup
Create a new `.env` file in the root directory, using the `.env.example` file for reference.

## Running
```sh
go run main.go
```

## Routes
|Method|Path|Description|
|-|-|-|
|GET|`/projects`|Get many projects|
|POST|`/projects`|Create one project|
|GET|`/projects/:pid`|Get project by ID|
|POST|`/projects/:pid`|Update project by ID|
|GET|`/projects/:pid/access_grant`|Get a project-scoped access grant for Storj|
|GET|`/projects/:pid/branches`|Get many branches for a project|
|POST|`/projects/:pid/branches`|Create one branch for a project|
|GET|`/projects/:pid/branches/default`|Get the default branch of a project|
|GET|`/projects/:pid/branches/:bid_or_name`|Get one branch by ID or name for a project|
|DELETE|`/projects/:pid/branches/:bid_or_name`|Delete one branch by ID or name for a project|
|GET|`/projects/:pid/branches/:bid/commits`|Get many commits for a branch|
|GET|`/projects/:pid/commits`|Get many commits for a project|
|POST|`/projects/:pid/commits`|Create one commit for a project|
|GET|`/projects/:pid/commits/index/:idx`|Get one commit by index for a project|
|GET|`/projects/:pid/commits/:cid`|Get one commit by ID for a project|
|GET|`/projects/:pid/commits/:cid`|Update one commit for a project|

### Terminology
|Term|Description|
|-|-|
|`pid`|Project ID|
|`bid`|Branch ID|
|`cid`|Commit ID|
