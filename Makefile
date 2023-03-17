## Simple makefile for goreleaser to build locally

all:
	goreleaser build --snapshot --clean --single-target
