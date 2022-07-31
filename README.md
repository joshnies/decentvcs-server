# DecentVCS

Core server for DecentVCS, the simple, affordable, and decentralized version control system.

## Requirements

| Dependency | Version |
| ---------- | ------- |
| `go`       | 1.18+   |

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

## Using the REST API

### Authentication

We use Stytch for session-based authentication. Once the user goes through one of Stytch's auth flows, you'll receive
a session token that can be sent with any request to the VCS server as the `X-Session-Token` header.

Example request:

```
POST /projects/myteam/myproject
X-Session-Token: ****1234
Content-Type: application/json
Body:
{
    "hello": "world"
}
```

### Routes

| Method | Path                                                               | Description                                      |
| ------ | ------------------------------------------------------------------ | ------------------------------------------------ |
| POST   | `/projects`                                                        | Create one project                               |
| GET    | `/projects/:team_name/:project_name`                               | Get one project                                  |
| PUT    | `/projects/:team_name/:project_name`                               | Update one project by ID                         |
| GET    | `/projects/:team_name/:project_name/branches`                      | Get many branches for a project                  |
| POST   | `/projects/:team_name/:project_name/branches`                      | Create one branch for a project                  |
| GET    | `/projects/:team_name/:project_name/branches/default`              | Get the default branch of a project              |
| GET    | `/projects/:team_name/:project_name/branches/:branch_name`         | Get one branch by ID or name for a project       |
| DELETE | `/projects/:team_name/:project_name/branches/:branch_name`         | Delete one branch by ID or name for a project    |
| POST   | `/projects/:team_name/:project_name/branches/:branch_name/commit`  | Create one commit                                |
| GET    | `/projects/:team_name/:project_name/branches/:branch_name/commits` | Get many commits for a branch                    |
| GET    | `/projects/:team_name/:project_name/commits`                       | Get many commits for a project                   |
| GET    | `/projects/:team_name/:project_name/commits/:commit_index`         | Get one commit for a project                     |
| PUT    | `/projects/:team_name/:project_name/commits/:commit_index`         | Update one commit for a project                  |
| GET    | `/projects/:team_name/:project_name/storage/presign/many`          | Presign many objects (`GET` method only)         |
| POST   | `/projects/:team_name/:project_name/storage/presign/:method`       | Presign one object                               |
| POST   | `/projects/:team_name/:project_name/storage/multipart/complete`    | Complete a multipart upload                      |
| DELETE | `/projects/:team_name/:project_name/storage/unused`                | Delete all unused files in storage for a project |
| GET    | `/teams`                                                           | Get many teams                                   |
| POST   | `/teams`                                                           | Create one team                                  |
| GET    | `/teams/:team_name`                                                | Get one team                                     |
| PUT    | `/teams/:team_name`                                                | Update one team                                  |
| DELETE | `/teams/:team_name`                                                | Delete one team                                  |
| GET    | `/users/:user_id`                                                  | Get one Stytch user                              |
