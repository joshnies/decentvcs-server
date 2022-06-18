# DecentVCS

Official server for DecentVCS.

## Requirements

| Dependency | Version |
| ---------- | ------- |
| `go`       | 1.18+   |

### Architecture

DecentVCS relies on certain architecture components. You're free to choose any of the supported providers below.

| Component    | Supported Providers                                        |
| ------------ | ---------------------------------------------------------- |
| SQL database | MySQL, PostgreSQL, SQLite, SQL Server                      |
| Auth service | [Auth0](https://auth0.com/), [Stytch](https://stytch.com/) |

> We're always working on adding support for more components, so stay tuned! Feel free to submit a PR if you're up to it as well 🤗

## Environment

#### Dotenv

Create a new `.env` file in the root directory, using the `.env.example` file for reference.

#### Doppler

We recommend using [Doppler](https://doppler.com) in deployed environments for enhanced security.

## Running (local development)

```sh
# With dotenv
go run main.go

# With doppler
./run
# or
doppler run -- go run main.go
```

## Build

```sh
./install.sh
```

It should be added to your path automatically. You can then run the built executable by running:

```sh
dvcs-server
```

## REST API Routes

| Method | Path                                   | Description                                   |
| ------ | -------------------------------------- | --------------------------------------------- |
| GET    | `/projects`                            | Get many projects                             |
| POST   | `/projects`                            | Create one project                            |
| GET    | `/projects/:owner_alias/:project_name` | Get one project by blob                       |
| GET    | `/projects/:pid`                       | Get one project by ID                         |
| POST   | `/projects/:pid`                       | Update one project by ID                      |
| GET    | `/projects/:pid/branches`              | Get many branches for a project               |
| POST   | `/projects/:pid/branches`              | Create one branch for a project               |
| GET    | `/projects/:pid/branches/default`      | Get the default branch of a project           |
| GET    | `/projects/:pid/branches/:bid_or_name` | Get one branch by ID or name for a project    |
| DELETE | `/projects/:pid/branches/:bid_or_name` | Delete one branch by ID or name for a project |
| GET    | `/projects/:pid/branches/:bid/commits` | Get many commits for a branch                 |
| GET    | `/projects/:pid/commits`               | Get many commits for a project                |
| POST   | `/projects/:pid/commits`               | Create one commit for a project               |
| GET    | `/projects/:pid/commits/index/:idx`    | Get one commit by index for a project         |
| GET    | `/projects/:pid/commits/:cid`          | Get one commit by ID for a project            |
| GET    | `/projects/:pid/commits/:cid`          | Update one commit for a project               |
| GET    | `/projects/storage/presign/many`       | Presign many objects (`GET` method only)      |
| POST   | `/projects/storage/presign/:method`    | Presign one object                            |
| POST   | `/projects/storage/multipart/complete` | Complete a multipart upload                   |

### Terminology

| Term  | Description |
| ----- | ----------- |
| `pid` | Project ID  |
| `bid` | Branch ID   |
| `cid` | Commit ID   |
