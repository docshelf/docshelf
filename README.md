# DocShelf
[![build](https://gitlab.com/docshelf/docshelf/badges/master/build.svg?job=test)](https://gitlab.com/docshelf/docshelf/pipelines) [![coverage](https://gitlab.com/docshelf/docshelf/badges/master/coverage.svg?job=test)](https://docshelf.gitlab.io/docshelf)
[![documentation](https://godoc.org/github.com/docshelf/docshelf?status.svg)](http://godoc.org/github.com/docshelf/docshelf)

A lightweight, team documentation solution that won't make you pull your hair out.

## !WIP!
This project is still a pre-alpha work in progress and isn't suitable for any real use cases yet. Come back soon though! :smile:

## Quickstart
The fastest way to get up and running with docshelf is to spin everything up with docker compose.
```
$ docker compose up
```
Navigating to [http://localhost:9001/](http://localhost:9001/) should pop up a login window.

Running with `docker-compose` runs the UI as a dev server, so bundles should be updated as you make changes  (although you will have to manually refresh your browser). The API also has rudimentary "hot reloading". Any time you modify a go file, a new build will be generated and the running binary will be replaced.

## Local Development
### API
To get the docshelf API running natively on your local machine, you just need to have the go compiler installed.
```
$ go run cmd/server/main.go
```
When docshelf starts up, it will spin up a local bolt database, a bleve search index, and all documents will be stored locally in a `documents/` folder.

### Experimental UI
There's curently a bare bones UI, written in svelte, that serves as a nicer way of testing docshelf features than running dozens of postman requests. You can run the dev server with npm.

```
$ cd ui/
$ npm install
$ npm run dev
```

#### Caddy reverse proxy
In order to serve both the UI and the API under the same domain (`localhost:9001` is the default), you can proxy through caddy. Download the binary release for your system from [the caddy download page](https://caddyserver.com/download).

From the root of the project:
```
$ caddy start
```

Once caddy, the UI dev server, and API are all running, you should be able to navigate to https://localhost:9001/ to load docshelf.

## Backends
### AWS
If you want to test docshelf with the AWS backends, all you have to do is set some environment variables. This assumes that your AWS credentials are already present in your environment.

#### Dynamo Backend
```
$ DS_BACKEND=dynamo go run cmd/server/main.go
```
Docshelf will automatically provision the necessary dynamo tables if they don't exist, so give it a minute or two on the first startup.


#### S3 File Store
```
$ DS_FILE_BACKEND=s3 DS_S3_BUCKET=docshelf-test go run cmd/server/main.go
```

## Configuration
Currently, docshelf can only be configured through environment variables. This table shows all of the current options that can be set.

| Var                     | Possible Values  | Description                                     |
|-------------------------|------------------|-------------------------------------------------|
| DS_BACKEND              | bolt, dynamo     | Backend for users, doc metadata, etc.           |
| DS_FILE_BACKEND         | disk, s3         | How to store document content                   |
| DS_TEXT_INDEX           | bleve, elastic\* | What text index to use for search               |
| DS_S3_BUCKET            | string           | The bucket to use with the s3 file backend      |
| DS_FILE_PREFIX          | string           | The path/prefix to apply to all saved documents |
| DS_HOST                 | string           | The host for the API to listen on               |
| DS_PORT                 | 0-65535          | The port for the API to listen on               |
| DS_GOOGLE_CLIENT_ID     | string           | The google client ID to use during oauth        |
| DS_GOOGLE_CLIENT_SECRET | string           | The google client secret to use during oauth    |
| DS_GITHUB_CLIENT_ID     | string           | The github client ID to use during oauth        |
| DS_GITHUB_CLIENT_SECRET | string           | The github client secret to use during oauth    |

_\*elastic is not currenlty supported, but will be in the near future_

More configuration options will become available as dochself becomes more full-featured.

