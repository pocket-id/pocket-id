.PHONY: default

default:
	docker build --platform linux/amd64 -t registry.digitalocean.com/vru/pocket-id-vru:latest . && docker push registry.digitalocean.com/vru/pocket-id-vru:latest
