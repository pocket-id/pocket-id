# Contributing

I am happy that you want to contribute to Pocket ID and help to make it better! All contributions are welcome, including issues, suggestions, pull requests and more.

## Getting started

You've found a bug, have suggestion or something else, just create an issue on GitHub and we can get in touch.

## Submit a Pull Request

Before you submit the pull request for review please ensure that

- The pull request naming follows the [Conventional Commits specification](https://www.conventionalcommits.org):

  `<type>[optional scope]: <description>`

  example:

  ```
  fix: hide global audit log switch for non admin users
  ```

  Where `TYPE` can be:
  - **feat** - is a new feature
  - **doc** - documentation only changes
  - **fix** - a bug fix
  - **refactor** - code change that neither fixes a bug nor adds a feature

- Your pull request has a detailed description
- You run `pnpm format` to format the code

## Development Environment

Pocket ID consists of a frontend and backend. In production the frontend gets statically served by the backend, but in development they run as separate processes to enable hot reloading.

There are two ways to get the development environment setup:

### 1. Install required tools

#### With Dev Containers

If you use [Dev Containers](https://code.visualstudio.com/docs/remote/containers) in VS Code, you don't need to install anything manually, just follow the steps below.

1. Make sure you have [Dev Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) extension installed
2. Clone and open the repo in VS Code
3. VS Code will detect .devcontainer and will prompt you to open the folder in devcontainer
4. If the auto prompt does not work, hit `F1` and select `Dev Containers: Open Folder in Container.`, then select the pocket-id repo root folder and it'll open in container.

#### Without Dev Containers

If you don't use Dev Containers, you need to install the following tools manually:

- [Node.js](https://nodejs.org/en/download/) >= 24
- [Go](https://golang.org/doc/install) >= 1.26
- [Git](https://git-scm.com/downloads)

### 2. Setup

#### Backend

The backend is built with [Gin](https://gin-gonic.com) and written in Go. To set it up, follow these steps:

1. Open the `backend` folder
2. Copy the `.env.development-example` file to `.env` and edit the variables as needed
3. Start the backend with `go run -tags exclude_frontend ./cmd`

### Frontend

The frontend is built with [SvelteKit](https://kit.svelte.dev) and written in TypeScript. To set it up, follow these steps:

1. Open the `pocket-id` project folder
2. Copy the `frontend/.env.development-example` file to `frontend/.env` and edit the variables as needed
3. Install the dependencies with `pnpm install`
4. Start the frontend with `pnpm dev`

You're all set! The application is now listening on `localhost:3000`. The backend gets proxied trough the frontend in development mode.

### Testing

We are using [Playwright](https://playwright.dev) for end-to-end testing.

If you are contributing to a new feature please ensure that you add tests for it. The tests are located in the `tests` folder at the root of the project.

The tests can be run like this:

1. Install the dependencies from the root of the project `pnpm install`

2. Visit the setup folder by running `cd tests/setup`

3. Start the test environment by running `docker compose up -d --build`

4. Go back to the test folder by running `cd ..`
5. Run the tests with `pnpm dlx playwright test` or from the root project folder `pnpm test`

If you make any changes to the application, you have to rebuild the test environment by running `docker compose up -d --build` again.
