package commonServe

import (
	"fmt"
	"os/exec"
)

// ExecWithVariousPhpInterpreters tries to auto-detect PHP versions by trying different PHP interpreters
func ExecWithVariousPhpInterpreters(cmd string) *exec.Cmd {
	return exec.Command("sh", "-c", fmt.Sprintf("./%s || php82 %s || php81 %s || php80 %s || php74 %s || php8.2 %s || php8.1 %s || php8.0 %s || php7.4 %s", cmd, cmd, cmd, cmd, cmd, cmd, cmd, cmd, cmd))
}
