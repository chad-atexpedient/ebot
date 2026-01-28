# Developing ebot

What follows is a rundown on different ways to run and develop ebot, its UI and its tools locally.

## Running ebot

The easiest way to run ebot locally is to run `make dev`. This will launch three processes: the API server, admin UI, and user UI. Opening `http://localhost:8080/admin/` will launch the admin UI. Changing the UI code will update the UI automatically. Changing any of the Go code requires restarting the server.

## Building and Running the ebot Docker Image

ebot is ultimately packaged into an image for distribution. You can build said image with `docker build -t my-ebot .`, and then run the image via `docker run -p 8080:8080 my-ebot`.

## Debugging ebot

It is possible to run the server and/or UIs in and IDE for debugging purposes. These steps layout what is necessary for JetBrains IDEs, but an equivalent process can be used with VSCode-based editors.

### Server

To run the server in GoLand:
1. Create a new "Go Build" configuration.
2. In the "Program Arguments" section, enter `server --dev-mode`.

Then you're ready to run or debug this target.

### User UI

To run the User UI in GoLand or WebStorm:
1. Create a new "npm" build.
2. In the "package.json" dropdown, select the `package.json` file in the `ui/user` directory.
3. In the "Command" dropdown, select `run`.
4. In the "Scripts" dropdown, select `dev`.
5. In the "Environment" section, enter `VITE_API_IN_BROWSER=true`.

Then you're ready to run or debug this target.

## Developing ebot Tools

ebot has a set of packaged tools. These tools are in the repo `github.com/obot-platform/tools`. By default, ebot will pull the tools from this repo. However, when developing tools in this repo, you can follow these steps to use a local copy.

1. Clone `github.com/obot-platform/tools` to your local machine.
2. In the root directory of the tools repo on your local machine, run `make build`.
3. Run the ebot server, either with `make dev` or in your IDE, with the `GPTSCRIPT_TOOL_REMAP` environment variable set to `github.com/obot-platform/tools=<local-tools-fork-root-directory>`; e.g. If you cloned the tools repo to the directory "above" the ebot repo, you'd use `GPTSCRIPT_TOOL_REMAP='github.com/obot-platform/tools=../tools' make dev`.

Now, any time one of these tools is run, your local copy will be used.

> [!IMPORTANT]
> Any time you change a Go based tool in your local repo, you must run `make build` in the tools repo for the changes to take effect with ebot.

> [!NOTE]
> Tool definitions and metadata are only synced to ebot every hour. Therefore, if you make a change to the tool in your local machine, it may not reflect immediately in ebot. Rest assured that the latest version is used when running the tool.

## ebot Server Dev Mode

In the description above for running the server in an IDE, the `--dev-mode` flag is used. This flag is also used when running the server with `make dev`. This does a few things (like turns on debug logging), the most helpful of which is to give you access to the database via `kubectl`. The kubeconfig is located at `tools/devmode-kubeconfig`.

For example, from the root directory of the ebot repo, you can list all agents in your setup with `kubectl --kubeconfig tools/devmode-kubeconfig get agents`.

## ebot Credentials

The GPTScript credentials for ebot are, by default, stored in a SQLite database called `ebot-credentials.db` in the root of the ebot repo. You can use the `sqlite3` CLI to inspect the database directly: `sqlite3 ebot-credentials.db`.

## Resetting

There may be times when you want to completely wipe your setup and start fresh. The location of data and caches is dependent on your system. For Mac or Linux, you can run the respective command in the root of the ebot repo on your local machine.

On Mac:
```bash
rm -rf ~/Library/Application\ Support/ebot &&
rm -rf ~/Library/Application\ Support/gptscript &&
rm -rf ~/Library/Caches/ebot &&
rm -rf ~/Library/Caches/gptscript &&
rm ebot.db ebot-credentials.db
```

On Linux:
```bash
rm -rf ~/.local/share/ebot &&
rm -rf ~/.local/share/gptscript &&
rm -rf ~/.cache/ebot &&
rm -rf ~/.cache/gptscript &&
rm ebot.db ebot-credentials.db
```

## Serving the Documentation

The documentation for ebot is in the main repo. You can serve the documentation from your local machine by running `make serve-docs` in the root of the ebot repo.

## Other Configuration

ebot is configured via environment variables. You can see the relevant environment variables by building the binary (as above) and running `./bin/ebot server --help`. There is also documentation available. You can serve the documentation locally as above.
