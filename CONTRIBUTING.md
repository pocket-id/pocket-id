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
  feat(share): add password protection
  ```

  Where `TYPE` can be:

  - **feat** - is a new feature
  - **doc** - documentation only changes
  - **fix** - a bug fix
  - **refactor** - code change that neither fixes a bug nor adds a feature

- Your pull request has a detailed description
- You run `npm run format` to format the code

## Setup project

Pocket ID consists of a frontend, backend and a reverse proxy. There are two ways to get the development environment setup:

## 1. Using DevContainers

1. Make sure you have [Dev Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) extension installed
2. Clone and open the repo in VS Code
3. VS Code will detect .devcontainer and will prompt you to open the folder in devcontainer
4. If the auto prompt does not work, hit `F1` and select `Dev Containers: Open Folder in Container.`, then select the pocket-id repo root folder and it'll open in container.

## 2. Manual

### Backend

The backend is built with [Gin](https://gin-gonic.com) and written in Go.

#### Setup

1. Open the `backend` folder
2. Copy the `.env.example` file to `.env` and edit the variables as needed
3. Start the backend with `go run -tags e2etest,exclude_frontend ./cmd`

### Frontend

The frontend is built with [SvelteKit](https://kit.svelte.dev) and written in TypeScript.

#### Setup

1. Open the `frontend` folder
2. Copy the `.env.example` file to `.env` and edit the variables as needed
3. Install the dependencies with `npm install`
4. Start the frontend with `npm run dev`

You're all set! The application is now listening on `localhost:3000`. The backend gets proxied trough the frontend in development mode.

### Testing

We are using [Playwright](https://playwright.dev) for end-to-end testing.

The tests can be run like this:

1. Start the backend normally
2. Start the frontend in production mode with `npm run build && node --env-file=.env build/index.js`
3. Run the tests with `npm run test`
