.PHONY: run vis plot clean gen

gen:
	@go run cmd/generator/main.go configs/$(config).yaml

run:
	@go run cmd/main.go

vis:
	@node visualizer/server.js

plot:
	@.venv/bin/python plotter/main.py

clean:
	@rm -rf data/metrics/*
	@rm -rf data/topology/*