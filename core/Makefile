# DO NOT EDIT MANUALLY
# generated by build.sh

GO_DEPS := subsystem/perfmonger_darwin.go subsystem/perfmonger.go subsystem/perfmonger_linux.go subsystem/perfmonger_linux_test.go subsystem/stat.go subsystem/stat_test.go subsystem/usage.go subsystem/usage_test.go utils.go

.PHONY: all build clean

all: build


../lib/exec/perfmonger-recorder_linux_amd64: cmd/perfmonger-recorder/perfmonger-recorder.go $(GO_DEPS)
	cd cmd/perfmonger-recorder && go build -o ../../$@ perfmonger-recorder.go


../lib/exec/perfmonger-player_linux_amd64: cmd/perfmonger-player/perfmonger-player.go $(GO_DEPS)
	cd cmd/perfmonger-player && go build -o ../../$@ perfmonger-player.go


../lib/exec/perfmonger-viewer_linux_amd64: cmd/perfmonger-viewer/perfmonger-viewer.go $(GO_DEPS)
	cd cmd/perfmonger-viewer && go build -o ../../$@ perfmonger-viewer.go


../lib/exec/perfmonger-summarizer_linux_amd64: cmd/perfmonger-summarizer/perfmonger-summarizer.go $(GO_DEPS)
	cd cmd/perfmonger-summarizer && go build -o ../../$@ perfmonger-summarizer.go


../lib/exec/perfmonger-plot-formatter_linux_amd64: cmd/perfmonger-plot-formatter/perfmonger-plot-formatter.go $(GO_DEPS)
	cd cmd/perfmonger-plot-formatter && go build -o ../../$@ perfmonger-plot-formatter.go


../lib/exec/perfmonger-recorder_darwin_amd64: cmd/perfmonger-recorder/perfmonger-recorder.go $(GO_DEPS)
	cd cmd/perfmonger-recorder && go build -o ../../$@ perfmonger-recorder.go


../lib/exec/perfmonger-player_darwin_amd64: cmd/perfmonger-player/perfmonger-player.go $(GO_DEPS)
	cd cmd/perfmonger-player && go build -o ../../$@ perfmonger-player.go


../lib/exec/perfmonger-viewer_darwin_amd64: cmd/perfmonger-viewer/perfmonger-viewer.go $(GO_DEPS)
	cd cmd/perfmonger-viewer && go build -o ../../$@ perfmonger-viewer.go


../lib/exec/perfmonger-summarizer_darwin_amd64: cmd/perfmonger-summarizer/perfmonger-summarizer.go $(GO_DEPS)
	cd cmd/perfmonger-summarizer && go build -o ../../$@ perfmonger-summarizer.go


../lib/exec/perfmonger-plot-formatter_darwin_amd64: cmd/perfmonger-plot-formatter/perfmonger-plot-formatter.go $(GO_DEPS)
	cd cmd/perfmonger-plot-formatter && go build -o ../../$@ perfmonger-plot-formatter.go


build: ../lib/exec/perfmonger-recorder_linux_amd64 ../lib/exec/perfmonger-player_linux_amd64 ../lib/exec/perfmonger-viewer_linux_amd64 ../lib/exec/perfmonger-summarizer_linux_amd64 ../lib/exec/perfmonger-plot-formatter_linux_amd64 ../lib/exec/perfmonger-recorder_darwin_amd64 ../lib/exec/perfmonger-player_darwin_amd64 ../lib/exec/perfmonger-viewer_darwin_amd64 ../lib/exec/perfmonger-summarizer_darwin_amd64 ../lib/exec/perfmonger-plot-formatter_darwin_amd64

clean:
	rm -f ../lib/exec/perfmonger-recorder_linux_amd64 ../lib/exec/perfmonger-player_linux_amd64 ../lib/exec/perfmonger-viewer_linux_amd64 ../lib/exec/perfmonger-summarizer_linux_amd64 ../lib/exec/perfmonger-plot-formatter_linux_amd64 ../lib/exec/perfmonger-recorder_darwin_amd64 ../lib/exec/perfmonger-player_darwin_amd64 ../lib/exec/perfmonger-viewer_darwin_amd64 ../lib/exec/perfmonger-summarizer_darwin_amd64 ../lib/exec/perfmonger-plot-formatter_darwin_amd64

