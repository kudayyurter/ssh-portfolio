package shell

import (
	"fmt"
	"strings"
)

func init() {
	register("clear", command{run: runClear, group: "Utilities", usage: "clear", about: "clear the screen"})
	register("history", command{run: runHistory, group: "Utilities", usage: "history", about: "commands you've run"})
	register("echo", command{run: runEcho, group: "Utilities", usage: "echo [text]", about: "print text"})
	register("date", command{run: runDate, group: "Utilities", usage: "date", about: "current time (UTC)"})
	register("exit", command{run: runExit, group: "Utilities", usage: "exit", about: "leave"})
	register("logout", command{run: runExit})
	register("boot", command{run: runBoot, group: "Utilities", usage: "boot", about: "replay the startup animation"})

	register("sudo", command{run: func(*Session, []string) Result {
		return fail("guest is not in the sudoers file. This incident will be reported.")
	}})
	for _, name := range []string{"rm", "mv", "cp", "touch", "mkdir"} {
		register(name, command{run: readOnly(name)})
	}
	for _, name := range []string{"vim", "vi", "nano", "emacs"} {
		register(name, command{run: editor(name)})
	}
}

func runClear(*Session, []string) Result { return Result{Action: ActionClear} }

func runHistory(s *Session, _ []string) Result {
	lines := make([]string, len(s.history))
	for i, h := range s.history {
		lines[i] = fmt.Sprintf("%5d  %s", i+1, h)
	}
	return ok(strings.Join(lines, "\n"))
}

func runEcho(_ *Session, args []string) Result { return ok(strings.Join(args, " ")) }

func runDate(s *Session, _ []string) Result {
	return ok(s.now().UTC().Format("Mon Jan _2 15:04:05 UTC 2006"))
}

func runExit(*Session, []string) Result { return Result{Output: "logout", Action: ActionExit} }

func runBoot(s *Session, _ []string) Result {
	if !s.interactive {
		return fail("boot: needs an interactive terminal (try ssh -t)")
	}
	return Result{Action: ActionBoot}
}

func readOnly(name string) func(*Session, []string) Result {
	return func(*Session, []string) Result {
		return fail(name + ": read-only file system (nice try)")
	}
}

func editor(name string) func(*Session, []string) Result {
	return func(_ *Session, args []string) Result {
		target := "<file>"
		if len(args) > 0 {
			target = args[0]
		}
		return fail(name + ": read-only file system — try cat " + target)
	}
}
