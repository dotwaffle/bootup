//go:build tools

// Package tools pins tool-time packages that are compiled by helper scripts.
package tools

import (
	_ "github.com/u-root/u-root/cmds/boot/boot"
	_ "github.com/u-root/u-root/cmds/core/cat"
	_ "github.com/u-root/u-root/cmds/core/gosh"
	_ "github.com/u-root/u-root/cmds/core/init"
	_ "github.com/u-root/u-root/cmds/core/insmod"
	_ "github.com/u-root/u-root/cmds/core/ip"
	_ "github.com/u-root/u-root/cmds/core/ls"
	_ "github.com/u-root/u-root/cmds/core/mount"
	_ "github.com/u-root/u-root/cmds/core/wget"
)
