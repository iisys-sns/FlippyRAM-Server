# FlippyRAM-Server
flippyram-server is a basic webserver written in Go. It can be used to receive datasets collected with the [FLIPPYR.AM](https://github.com/iisys-sns/FlippyRAM) ISO image.
The current version of flippyram-server shows the same website hosted at [flippyr.am](https://flippyr.am).
We want to thank [Claudia](https://github.com/cehter) for developing the website.

You can use the server as-is **internally** within your organization, for example to test computer systems on your campus.
Adjust the content of `flippyram/web/` (fix links, change logo, adjust text, etc.) before making the application **publically available**, though.

## Repository structure
* `main.go` and  `src/`: Source code of the server (used to compile the binary)
* `bin/flippyram-server`: Compiled binary executable of the server
* `flippyram-server.service`: Systemd unit that can be used to run flippyram-server as service
* `config.json`: Configuration file of the server (for local testing, uploaded as default config when no config exists)
* `flippyram/data`: Directory used to upload ZIP files (for local testing)
* `flippyram/flippyram.crt`,  `flippyram/flippyram.key`: TLS certificate and key (for local testing). Run `init.sh` to generate them.
* `flippyram/permitted_uploads.txt`: List of files permitted within uploaded ZIP files (for local testing and deployment)
* `flippyram/whitelist.txt`: List of files permitted to be served by the webservice (for local testing and deployment)
* `flippyram/web/`: Directory that contains the files used by the webservice (must be added to `flippyram/whitelist.txt` to be served locally and deployed to remote systems)
* `deploy.sh`: Script to deploy the local version to a remove server (see section on `Deployment` below for more information)
* `init.sh`: Script to initialize the local repository (see section `Preparation` below for more information)

## Preparation
To run flippyram-server locally, execute the `init.sh` script once to generate a key and certificate used for TLS.
If you already have a key and certificate you want to use, you can skip this step.

To initialize the Go repository, run
```
go mod init flippyram-server
go mod tidy
```

## Build flippyram-server
Just type
```
make
```
to build the flippyram-server binary locally.

## Run flippyram-server
To run flippyram-server locally, adjust the `config.json` file.
Use the following command afterwards to start the server:
```
./bin/flippyram-server config.json
```

If the config was not modified, the server will listen on `127.0.0.1:8443`.

## Deployment
To deploy flippyram-server to a remote server, you can use the `deploy.sh` script.
Just run the following command:
```
sh deploy.sh <hostname-or-ip-address> [<mode>]
```
to deploy the current version of flippyram-server to the server.

There are three different modes: `WHITELIST`, `SERVER`, `PARTIAL`, and `FULL`.

For the initial deployment, use `FULL` as mode.

### Server
In `SERVER` mode, the script will perform the following steps:
* Create a new service user if not already existing
* Check if there is already a config file and upload the local file if there is no config file on the server
* Stop the flippyram-server service
* Upload the binary and systemd Unit file (overwrites existing files)
* Adjust permissions of all files
* Reload systemd units
* Start the flippyram-server service

### Content
In `CONTENT` mode, the script will perform the following steps:
* Read the remote config to copy the following files to the correct locations
* Create the directory used to upload the datasets
* Create the directory used for the files related to the webservice
* Upload all files that are on the whitelist for the webservice (e.g., upload only whitelisted files)
* Restart the flippyram-server service to reload all files

### Whitelist
In `WHITELIST` mode, the script will perform the following steps:
* Read the remote config to copy the following files to the correct locations
* Upload the whitelist of the files permitted to be uploaded within the ZIP archives
* Restart the flippyram-server service to reload all files

### Full
In `FULL` mode, the script will perform the steps of `SERVER` mode followed by
the steps of `CONTENT` mode followed by the steps of `WHITELIST` mode.

## Finishing the setup
### Databse
For running flippyram-server, a database is required to store valid tokens.
You can use `init.sql` to create the database.
Afterwards, adjust `config.json` and put the access details of the database there.

### Final steps
After the database is running, check all settings in `config.json` and adjust accordingly.
Finally, run `systemctl enable --now flippyram-server.service` to enable and start the systemd unit.
Afterwards, the server should be reachable.
