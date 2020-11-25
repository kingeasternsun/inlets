SRC = $(wildcard *.go)
GITVISION = `git describe`"-"`git rev-parse --short HEAD`
all:inletserver
inletserver:$(SRC)
	go build -o inletserver -x -ldflags  " -w -s -X main.BuildVersion=$(GITVISION)" $^
clean:
	rm -rvf inletserver

