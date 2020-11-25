SRC = $(wildcard *.go)
all:inletcs
inletcs:$(SRC)
	go build -o inletcs -x -ldflags  " -w -s " $^
clean:
	rm -rvf inletcs