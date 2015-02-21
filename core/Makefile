
GOSRC :=                                        \
        subsystem/perfmonger.go                 \
        subsystem/perfmonger_linux.go           \
        subsystem/stat.go                       \
        subsystem/usage.go

all: perfmonger-recorder perfmonger-player

perfmonger-recorder: perfmonger-recorder.go $(GOSRC)
	go build $<


perfmonger-player: perfmonger-player.go $(GOSRC)
	go build $<

