#!/bin/bash

usage() {
	echo "Usage: "$0" <server> [<mode>]"
	echo "  <server>: IP address or hostname of the server where the system should be deployed to"
	echo "  <mode>: Mode of the deployment. Can be:"
	echo "    WHITELIST: Deploy only whitelist of files permitted within uploaded ZIP archives (default)"
	echo "    SERVER: Deploy only server components (config file if not existing, systemd unit, server binary)"
	echo "    CONTENT: Deploy only data within the 'flippyram' directory (excluding the uploaded ZIP whitelist, see WHITELIST mode)"
	echo "    FULL: Perform SERVER, CONTENT, and WHITELIST mode after each other"
	exit
}

getJsonField() {
	config="$1"
	field="$2"
	echo "$config" | grep "$field" | cut -d ":" -f 2 | sed 's/ "\(.*\)",*$/\1/g'
}

if [ $# -ne 1 ] && [ $# -ne 2 ]; then
	usage
fi

getDirectoryFromFile() {
	filePath="$1"
	echo "$filePath" | rev | cut -d "/" -f 2- | rev
}

SERVER="$1"
SERVER_MODE=0
CONTENT_MODE=0
WHITELIST_MODE=1

if [ $# -eq 2 ]; then
	if [ "$2" = "SERVER" ]; then
		WHITELIST_MODE=0
		SERVER_MODE=1
	elif [ "$2" = "CONTENT" ]; then
		WHITELIST_MODE=0
		CONTENT_MODE=1
	elif [ "$2" = "FULL" ]; then
		SERVER_MODE=1
		CONTENT_MODE=1
	elif [ "$2" != "WHITELIST" ]; then
		usage
	fi
fi

if [ $SERVER_MODE -eq 1 ]; then
	ssh "root@$SERVER" useradd -s /bin/nologin -M flippyram-server
	ssh "root@$SERVER" mkdir /etc/flippyram-server/

	ssh "root@$SERVER" test -f /etc/flippyram-server/config.json
	if [ $? -ne 0 ]; then
		scp config.json "root@$SERVER:/etc/flippyram-server/"
	fi
	ssh "root@$SERVER" chmod 0750 /etc/flippyram-server
	ssh "root@$SERVER" chmod 0640 /etc/flippyram-server/config.json
	ssh "root@$SERVER" chown -R root:flippyram-server /etc/flippyram-server

	ssh "root@$SERVER" systemctl stop flippyram-server
	scp bin/flippyram-server "root@$SERVER:/usr/bin/"
	ssh "root@$SERVER" chmod 0750 /usr/bin/flippyram-server
	ssh root@$SERVER chown root:flippyram-server /usr/bin/flippyram-server

	scp flippyram-server.service "root@$SERVER:/etc/systemd/system/"
	ssh "root@$SERVER" chmod 0600 /etc/systemd/system/flippyram-server.service
	ssh "root@$SERVER" systemctl daemon-reload
	ssh "root@$SERVER" systemctl enable --now flippyram-server
	ssh "root@$SERVER" systemctl restart flippyram-server
fi

if [ $CONTENT_MODE -eq 1 ]; then
	config=$(ssh root@$SERVER cat /etc/flippyram-server/config.json)

	datasetDirectory="$(getJsonField "$config" "dataset_directory")"
	ssh "root@$SERVER" mkdir -p "$datasetDirectory"
	ssh "root@$SERVER" chmod 0770 "$datasetDirectory"
	ssh root@$SERVER chown -R root:flippyram-server "$datasetDirectory"

	webDirectory="$(getJsonField "$config" "web_root")"
	ssh "root@$SERVER" mkdir -p "$webDirectory"
	ssh "root@$SERVER" chmod 0750 "$webDirectory"
	ssh root@$SERVER chown -R root:flippyram-server "$webDirectory"

	whitelistPath="$(getJsonField "$config" "whitelist_path")"
	ssh "root@$SERVER" mkdir -p "$(getDirectoryFromFile "$whitelistPath")"
	scp flippyram/whitelist.txt "root@$SERVER:$whitelistPath"
	ssh "root@$SERVER" chmod 0640 "$whitelistPath"
	ssh "root@$SERVER" chown root:flippyram-server "$whitelistPath"

	for line in $(cat flippyram/whitelist.txt); do
		file=$(echo $line | cut -d "|" -f 1)
		if [ -f flippyram/web/$file ]; then
			for (( i=1; i<5; i++ )); do
				part="$(echo "$file" | cut -d "/" -f 1-${i})"
				if [[ "$part" == *"."* ]]; then
					break
				fi
				ssh "root@$SERVER" mkdir -p "$webDirectory${part}"
				ssh "root@$SERVER" chmod 0750 "${webDirectory}${part}"
				ssh "root@$SERVER" chown root:flippyram-server "${webDirectory}${part}"
			done
			scp "flippyram/web/$file" "root@$SERVER:$webDirectory${file}"
			ssh "root@$SERVER" chmod 0640 "${webDirectory}${file}"
			ssh "root@$SERVER" chown root:flippyram-server "${webDirectory}${file}"
		fi
	done

	ssh "root@$SERVER" systemctl restart flippyram-server
fi

if [ $WHITELIST_MODE -eq 1 ]; then
	config=$(ssh root@$SERVER cat /etc/flippyram-server/config.json)

	permittedUploadsPath="$(getJsonField "$config" "permitted_uploads_path")"
	ssh "root@$SERVER" mkdir -p "$(getDirectoryFromFile "$permittedUploadsPath")"
	scp flippyram/permitted_uploads.txt "root@$SERVER:$permittedUploadsPath"
	ssh "root@$SERVER" chmod 0640 "$permittedUploadsPath"
	ssh "root@$SERVER" chown root:flippyram-server "$permittedUploadsPath"

	ssh "root@$SERVER" systemctl restart flippyram-server
fi
