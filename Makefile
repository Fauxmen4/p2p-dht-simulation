.PHONY: vis

run:
	@go run cmd/main.go

vis:
	@node visualizer/server.js
	