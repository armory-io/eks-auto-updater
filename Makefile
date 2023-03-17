## Simple makefile for goreleaser to build locally

all:
	goreleaser release  --snapshot --clean --skip-publish
