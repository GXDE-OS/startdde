/*
 * Copyright (C) 2014 ~ 2018 Deepin Technology Co., Ltd.
 *
 * Author:     jouyouyun <jouyouwen717@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import "C"
import (
	"flag"
	"os"
	"syscall"

	"github.com/linuxdeepin/go-x11-client"
	"pkg.deepin.io/dde/startdde/display"
	"pkg.deepin.io/dde/startdde/iowait"
	"pkg.deepin.io/dde/startdde/watchdog"
	"pkg.deepin.io/dde/startdde/xsettings"
	"pkg.deepin.io/lib/dbus"
	"pkg.deepin.io/lib/gsettings"
	"pkg.deepin.io/lib/log"
	"pkg.deepin.io/lib/proxy"

	wl_display "pkg.deepin.io/dde/startdde/wl_display"
)

var logger = log.NewLogger("startdde")

var debug = flag.Bool("d", false, "debug")

var globalGSettingsConfig *GSettingsConfig

var globalCgExecBin string

var globalWmChooserLaunched bool

var globalXSManager *xsettings.XSManager

var _useWayland bool

func reapZombies() {
	// We must reap children process even we hasn't create anyone at this moment,
	// Because the startdde may be launched by exec syscall
	// in another existed process, like /usr/sbin/lighdm-session does.
	// NOTE: Don't use signal.Ignore(syscall.SIGCHILD), otherwise os/exec wouldn't work properly.
	//       And simply ignore SIGCHILD hasn't any helpful in here.
	for {
		pid, err := syscall.Wait4(-1, nil, syscall.WNOHANG, nil)
		if err != nil || pid == 0 {
			break
		}
	}
}

func shouldUseDDEKwin() bool {
	_, err := os.Stat("/usr/bin/kwin_no_scale")
	return err == nil
}

func main() {
	globalGSettingsConfig = getGSettingsConfig()
	reapZombies()

	// init x conn
	xConn, err := x.NewConn()
	if err != nil {
		logger.Warning(err)
		os.Exit(1)
	}

	flag.Parse()
	initSoundThemePlayer()

	tryMatchVM()
	go playLoginSound()

	err = gsettings.StartMonitor()
	if err != nil {
		logger.Warning("gsettings start monitor failed:", err)
	}
	proxy.SetupProxy()

	if os.Getenv("WAYLAND_DISPLAY") != "" {
		logger.Info("in wayland mode")
		_useWayland = true
		// 相比于 X11 环境，在 Wayland 环境下，先于启动核心组件之前启动了 wl_display 模块。
		err = wl_display.Start()
		if err != nil {
			logger.Warning(err)
		}
		recommendedScaleFactor = wl_display.GetRecommendedScaleFactor()
	}
	else {
		// 使用 X11 环境时, 把 display 模块的启动分成两个部分，前一部分在 core components 启动之前启动，
		// 后一部分在 core components 启动之后启动。
		err = display.Start()
		if err != nil {
			logger.Warning("start display part1 failed:", err)
		}
	}

	if err != nil {
		logger.Warning(err)
	}

	xsManager, err := xsettings.Start(xConn, logger,
		display.GetRecommendedScaleFactor())
	if err != nil {
		logger.Warning(err)
	} else {
		globalXSManager = xsManager
	}
	go func() {
		inVM, _ := isInVM()
		if inVM {
			logger.Debug("try to correct vm resolution")
			correctVMResolution()
		}
	}()

	useKwin := shouldUseDDEKwin()

	sessionManager := startSession(xConn, useKwin)
	var getLockedFn func() bool
	if sessionManager != nil {
		getLockedFn = sessionManager.getLocked
	}
	watchdog.Start(getLockedFn, useKwin)

	if globalGSettingsConfig.iowaitEnabled {
		go iowait.Start(logger)
	} else {
		logger.Info("iowait disabled")
	}

	dbus.Wait()
}

func doSetLogLevel(level log.Priority) {
	logger.SetLogLevel(level)
	if !_useWayland {
		display.SetLogLevel(level)
	} else {
		wl_display.SetLogLevel(level)
	}
	watchdog.SetLogLevel(level)
}
