#		 _                       _
#	  __| |_ __  ___   ___  __ _| |
#	 / _` | '_ \/ __| / __|/ _` | |
#	| (_| | | | \__ \_\__ | (_| | |
#	 \__,_|_| |_|___(_|___/\__, |_|
#							  |_|
#
# DNS.sql is a sqlite3 extension that allows you to query DNS using SQL
#
# @author Riyaz Ali <me@riyazali.net>

VERSION=0.1.2
SRCS=$(wildcard *.go)
PREFIX=.build

$(PREFIX):
	mkdir -p $(PREFIX)

clean:
	-rm -rf $(PREFIX)

# ~~~~~~~~~~~~~~~~~~~~~~
# Targets that build shared object / dynamic library of the sqlite extension

$(PREFIX)/sqlite-dns-darwin-x64.dylib: $(SRCS) cmd/shared/shared.go
	CGO_ENABLED=1 CGO_CFLAGS="-DUSE_LIBSQLITE3" GOOS=darwin GOARCH=amd64 \
	CC="clang -arch x86_64" go build -trimpath -ldflags="-s -w" -buildmode=c-shared -o $@ cmd/shared/shared.go

$(PREFIX)/sqlite-dns-darwin-arm64.dylib: $(SRCS) cmd/shared/shared.go
	CGO_ENABLED=1 CGO_CFLAGS="-DUSE_LIBSQLITE3" GOOS=darwin GOARCH=arm64 \
	CC="clang -arch arm64" go build -trimpath -ldflags="-s -w" -buildmode=c-shared -o $@ cmd/shared/shared.go

# Target to build universal mac binary (https://developer.apple.com/documentation/apple-silicon/building-a-universal-macos-binary)
$(PREFIX)/sqlite-dns-darwin-universal.dylib: $(PREFIX)/sqlite-dns-darwin-x64.dylib $(PREFIX)/sqlite-dns-darwin-arm64.dylib
	lipo -create -output $@ $^

$(PREFIX)/sqlite-dns-linux-x64.so: $(SRCS) cmd/shared/shared.go
	CGO_ENABLED=1 CGO_CFLAGS="-DUSE_LIBSQLITE3" GOOS=linux GOARCH=amd64 \
	CC="zig cc -target x86_64-linux-gnu" go build -trimpath -ldflags="-s -w" -buildmode=c-shared -o $@ cmd/shared/shared.go

$(PREFIX)/sqlite-dns-linux-arm64.so: $(SRCS) cmd/shared/shared.go
	CGO_ENABLED=1 CGO_CFLAGS="-DUSE_LIBSQLITE3" GOOS=linux GOARCH=arm64 \
	CC="zig cc -target aarch64-linux-gnu" go build -trimpath -ldflags="-s -w" -buildmode=c-shared -o $@ cmd/shared/shared.go

# requires: https://github.com/ziglang/zig/issues/10989
# $(PREFIX)/sqlite-dns-windows-amd64.dll: $(SRCS) cmd/shared/shared.go
#  	CGO_ENABLED=1 CGO_CFLAGS="-DUSE_LIBSQLITE3" GOOS=windows GOARCH=amd64 \
# 	CC="zig cc -target x86_64-windows-msvc" go build -trimpath -ldflags="-s -w" -buildmode=c-shared -o $@ cmd/shared/shared.go

# Target to build all variants of the extension
extensions: $(PREFIX)/sqlite-dns-darwin-universal.dylib $(PREFIX)/sqlite-dns-linux-x64.so $(PREFIX)/sqlite-dns-linux-arm64.so

# ~~~~~~~~~~~~~~~~~~~~~~
# Targets that build files for bindings

$(PREFIX)/sqlite-dns-${VERSION}.tgz: extensions
	cp .build/*.{so,dylib} bindings/node/lib
	npm pack --pack-destination $(PREFIX) ./bindings/node

$(PREFIX)/wheels/sqlite_dns-${VERSION}-py3-none-manylinux2014_x86_64.whl: $(PREFIX)/sqlite-dns-linux-x64.so
	. .venv/bin/activate
	-rm -rf bindings/python/build bindings/python/sqlite_dns.egg-info
	cp $< bindings/python/sqlite_dns/lib/sqlite_dns.so
	(cd bindings/python && python setup.py bdist_wheel -b $(shell mktemp -d) -p manylinux2014_x86_64 -d ../../$(PREFIX)/wheels)
	-rm -rf bindings/python/sqlite_dns/lib/sqlite_dns.so bindings/python/build bindings/python/sqlite_dns.egg-info

$(PREFIX)/wheels/sqlite_dns-${VERSION}-py3-none-manylinux2014_aarch64.whl: $(PREFIX)/sqlite-dns-linux-arm64.so
	. .venv/bin/activate
	-rm -rf bindings/python/build bindings/python/sqlite_dns.egg-info
	cp $< bindings/python/sqlite_dns/lib/sqlite_dns.so
	(cd bindings/python && python setup.py bdist_wheel -b $(shell mktemp -d) -p manylinux2014_aarch64 -d ../../$(PREFIX)/wheels)
	-rm -rf bindings/python/sqlite_dns/lib/sqlite_dns.so bindings/python/build bindings/python/sqlite_dns.egg-info

$(PREFIX)/wheels/sqlite_dns-${VERSION}-py3-none-macosx_12_0_universal2.whl: $(PREFIX)/sqlite-dns-darwin-universal.dylib
	. .venv/bin/activate
	-rm -rf bindings/python/build bindings/python/sqlite_dns.egg-info
	cp $< bindings/python/sqlite_dns/lib/sqlite_dns.dylib
	(cd bindings/python && python setup.py bdist_wheel -b $(shell mktemp -d) -p macosx_12_0_universal2 -d ../../$(PREFIX)/wheels)
	-rm -rf bindings/python/sqlite_dns/lib/sqlite_dns.dylib bindings/python/build bindings/python/sqlite_dns.egg-info

wheels: $(PREFIX)/wheels/sqlite_dns-${VERSION}-py3-none-manylinux2014_x86_64.whl $(PREFIX)/wheels/sqlite_dns-${VERSION}-py3-none-manylinux2014_aarch64.whl \
	$(PREFIX)/wheels/sqlite_dns-${VERSION}-py3-none-macosx_12_0_universal2.whl

.PHONY: mac clean extensions wheels