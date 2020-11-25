SRC = $(wildcard *.go)
GITVISION = `git describe`"-"`git rev-parse --short HEAD`
all:inletclient
inletclient:$(SRC)
	go build -o inletclient -x -ldflags  " -w -s -X main.BuildVersion=$(GITVISION)" $^
clean:
	rm -rvf inletclient

