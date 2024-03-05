BINARY_NAME=go-mqttd
MAIN_VER=2.4.6

DIST_WINDOWS=_dist/${BINARY_NAME}.exe
DIST_LINUX=_dist/${BINARY_NAME}
DIST_ARM64=_dist/${BINARY_NAME}-arm64
DIST_MIPS64=_dist/${BINARY_NAME}-mips64

GO_VER=`go version | cut -d \  -f 3`
LDFLAGS="-s -w -X 'main.gover=${GO_VER}' -X 'main.cover=${MAIN_VER}'"

release: windows linux arm64 mips64
	@echo "copy files to server..."
	@scp -p ${DIST_WINDOWS} wlstl:/home/shares/archiving/v5release/luwakInstall/micro-services/bin
	@scp -p ${DIST_LINUX} wlstl:/home/shares/archiving/v5release/luwak_linux/programs
	@scp -p ${DIST_ARM64} wlstl:/home/shares/archiving/v5release/luwak_arm64/bin
	@scp -p ${DIST_MIPS64} wlstl:/home/shares/archiving/v5release/luwak_mips64/bin
	@echo "\nall done."

windows:
	@echo "building windows amd64 version..."
	@GOARCH=amd64 GOOS=windows CGO_ENABLED=0 go build -o ${DIST_WINDOWS} -ldflags=${LDFLAGS} main.go
	@echo "done.\n"

linux: modtidy
	@echo "building linux amd64 version..."
	@GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -o ${DIST_LINUX} -ldflags=${LDFLAGS} main.go
	@echo "done.\n"

arm64: modtidy
	@echo "building linux arm64/aarch64 version..."
	@GOARCH=arm64 GOOS=linux CGO_ENABLED=0 go build -o ${DIST_ARM64} -ldflags=${LDFLAGS} main.go
	@echo "done.\n"

mips64: modtidy
	@echo "building linux mips64 version..."
	@GOARCH=mips64 GOOS=linux CGO_ENABLED=0 go build -o ${DIST_MIPS64} -ldflags=${LDFLAGS} main.go
	@echo "done.\n"

modtidy:
	@go mod tidy

modupdate:
	@go get -u -v all
	@echo "done."

clean:
	@rm -fv _dist/${BINARY_NAME}*
