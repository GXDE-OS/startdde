PREFIX = /usr
GOPATH_DIR = gopath
GOPKG_PREFIX = pkg.deepin.io/dde/startdde
GOBUILD = go build -v
GOPATH="/usr/share/gocode-gxde/:/usr/share/gocode/:${CURDIR}/${GOPATH_DIR}"
all: build

prepare:
	@if [ ! -d ${GOPATH_DIR}/src/${GOPKG_PREFIX} ]; then \
		mkdir -p ${GOPATH_DIR}/src/$(dir ${GOPKG_PREFIX}); \
		ln -sf ../../../.. ${GOPATH_DIR}/src/${GOPKG_PREFIX}; \
		fi

auto_launch_json:
ifdef AUTO_LAUNCH_CHINESE
	cp misc/auto_launch/chinese.json misc/config/auto_launch.json
else
	cp misc/auto_launch/default.json misc/config/auto_launch.json
endif
	# check validity
	jq . misc/config/auto_launch.json >/dev/null

startdde:
	env GOPATH="${CURDIR}/${GOPATH_DIR}:${GOPATH}" ${GOBUILD} -o startdde

fix-xauthority-perm:
	env GOPATH="${CURDIR}/${GOPATH_DIR}:${GOPATH}" ${GOBUILD} -o fix-xauthority-perm ${GOPKG_PREFIX}/cmd/fix-xauthority-perm

build: prepare startdde auto_launch_json fix-xauthority-perm

test: prepare
	env GOPATH="${CURDIR}/${GOPATH_DIR}:${GOPATH}" go test -v ./...

install:
	install -Dm755 startdde ${DESTDIR}${PREFIX}/bin/startdde
	mkdir -p ${DESTDIR}${PREFIX}/share/xsessions
	#mkdir -p ${DESTDIR}${PREFIX}/share/wayland-sessions
	@for i in $(shell ls misc/xsessions/ | grep -E '*.in$$' );do sed 's|@PREFIX@|$(PREFIX)|g' misc/xsessions/$$i > ${DESTDIR}${PREFIX}/share/xsessions/$${i%.in}; done
	#@for i in $(shell ls misc/wayland-sessions/ | grep -E '*.in$$' );do sed 's|@PREFIX@|$(PREFIX)|g' misc/wayland-sessions/$$i > ${DESTDIR}${PREFIX}/share/wayland-sessions/$${i%.in}; done
	install -Dm755 misc/deepin-session ${DESTDIR}${PREFIX}/sbin/deepin-session
	#install -Dm755 misc/startdde_wayland ${DESTDIR}${PREFIX}/bin/startdde_wayland
	install -Dm755 fix-xauthority-perm ${DESTDIR}${PREFIX}/sbin/deepin-fix-xauthority-perm
	install -Dm644 misc/lightdm.conf ${DESTDIR}${PREFIX}/share/lightdm/lightdm.conf.d/60-deepin.conf
	install -Dm644 misc/startgxde ${DESTDIR}${PREFIX}/bin/startgxde
	mkdir -p ${DESTDIR}${PREFIX}/share/startdde/
	cp -f misc/config/* ${DESTDIR}${PREFIX}/share/startdde/
	mkdir -p ${DESTDIR}/etc/X11/Xsession.d/
	cp -f misc/00deepin-dde-env ${DESTDIR}/etc/X11/Xsession.d/

clean:
	-rm -rf ${GOPATH_DIR}
	-rm -f startdde
	-rm -rf obj-x86_64-linux-gnu/
	-rm -rf misc/config/auto_launch.json
	-rm -rf fix-xauthority-perm

rebuild: clean build
