build_dir=build
appname := yuquesync

sources := $(wildcard *.go)

build = GOOS=$(1) GOARCH=$(2) go build -o ${build_dir}/$(appname)-$(1)-$(2)
md5 = md5sum ${build_dir}/$(appname)-$(1)-$(2) > ${build_dir}/$(appname)-$(1)-$(2)_checksum.txt
tar =  tar -cvzf ${build_dir}/$(appname)-$(1)-$(2).tar.gz  -C ${build_dir}  $(appname)-$(1)-$(2) $(appname)-$(1)-$(2)_checksum.txt
delete = rm -rf ${build_dir}/$(appname)-$(1)-$(2) ${build_dir}/$(appname)-$(1)-$(2)_checksum.txt
ALL_LINUX = linux-amd64 \
	linux-386 \
	linux-arm \
	linux-arm64

ALL = $(ALL_LINUX) \
	darwin-amd64 \
	windows-amd64 

build_linux: $(ALL_LINUX:%=build/%)

build_all: $(ALL:%=build/%)

build/%:
	$(call build,$(firstword $(subst -, , $*)),$(word 2, $(subst -, ,$*)))
	$(call md5,$(firstword $(subst -, , $*)),$(word 2, $(subst -, ,$*)))
	$(call tar,$(firstword $(subst -, , $*)),$(word 2, $(subst -, ,$*)))
	$(call delete,$(firstword $(subst -, , $*)),$(word 2, $(subst -, ,$*)))
clean:
	rm -rf ${build_dir}/*

vet:
	go vet yuque.go