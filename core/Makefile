
GOSRC :=                                        \
        subsystem/perfmonger.go                 \
        subsystem/perfmonger_linux.go           \
        subsystem/stat.go                       \
        subsystem/usage.go

.PHONY: all clean

all: perfmonger-recorder perfmonger-player

clean:
	rm -f perfmonger-recorder perfmonger-player

perfmonger-recorder: perfmonger-recorder.go $(GOSRC)
	go build $<

perfmonger-player: perfmonger-player.go $(GOSRC)
	go build $<

