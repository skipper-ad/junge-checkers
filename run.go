package junge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

type result struct {
	status  Status
	public  string
	private string
}

// Run executes a checker action from os.Args and returns the exit code.
func Run(checker Checker) int {
	return RunWithArgs(checker, os.Args[1:], os.Stdout, os.Stderr)
}

// Main executes a checker action and exits the process. Use it from func main.
func Main(checker Checker) {
	os.Exit(Run(checker))
}

// RunWithArgs is useful for tests and custom launchers.
func RunWithArgs(checker Checker, args []string, stdout, stderr io.Writer) int {
	if checker == nil {
		res := result{
			status:  StatusCheckFailed,
			public:  "Checker failed",
			private: "checker is nil",
		}
		writeResult(stdout, stderr, res)
		return res.status.Code()
	}

	info, infoErr := checkerInfo(checker)
	if infoErr != nil {
		writeResult(stdout, stderr, *infoErr)
		return infoErr.status.Code()
	}
	timeout := time.Duration(info.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan result, 1)
	c := newC(ctx)

	go func() {
		defer func() {
			if v := recover(); v != nil {
				if _, ok := v.(checkerFinished); !ok {
					done <- result{
						status:  StatusCheckFailed,
						public:  "Checker failed",
						private: fmt.Sprintf("panic: %v", v),
					}
					return
				}
			}

			c.mu.Lock()
			defer c.mu.Unlock()
			if !c.finished {
				done <- result{
					status:  StatusCheckFailed,
					public:  "Checker failed",
					private: "checker did not report status",
				}
				return
			}
			done <- result{
				status:  c.status,
				public:  c.public,
				private: c.private,
			}
		}()

		runAction(c, checker, info, args)
	}()

	var res result
	select {
	case res = <-done:
	case <-ctx.Done():
		res = result{
			status:  StatusDown,
			public:  "Checker timed out",
			private: fmt.Sprintf("checker timeout after %s", timeout),
		}
	}

	writeResult(stdout, stderr, res)
	return res.status.Code()
}

func checkerInfo(checker Checker) (info *CheckerInfo, errResult *result) {
	defer func() {
		if v := recover(); v != nil {
			info = nil
			errResult = &result{
				status:  StatusCheckFailed,
				public:  "Checker failed",
				private: fmt.Sprintf("checker info panic: %v", v),
			}
		}
	}()
	return normalizeInfo(checker.Info()), nil
}

func runAction(c *C, checker Checker, info *CheckerInfo, args []string) {
	if len(args) == 0 {
		c.CheckFailed("Checker failed", "missing action")
	}

	switch action := args[0]; action {
	case "info":
		if len(args) != 1 {
			c.CheckFailedf("Checker failed", "info expects no arguments, got %d", len(args)-1)
		}
		data, err := json.Marshal(info)
		if err != nil {
			c.CheckFailed("Checker failed", fmt.Sprintf("marshal info: %v", err))
		}
		c.Finish(StatusOK, string(data), "")
	case "check":
		if len(args) != 2 {
			c.CheckFailedf("Checker failed", "check expects 1 argument: <host>, got %d", len(args)-1)
		}
		checker.Check(c, args[1])
	case "put":
		if len(args) != 5 {
			c.CheckFailedf("Checker failed", "put expects 4 arguments: <host> <flag_id> <flag> <vuln>, got %d", len(args)-1)
		}
		vuln, err := strconv.Atoi(args[4])
		if err != nil {
			c.CheckFailedf("Checker failed", "invalid vuln number %q", args[4])
		}
		checker.Put(c, args[1], args[2], args[3], vuln)
	case "get":
		if len(args) != 5 {
			c.CheckFailedf("Checker failed", "get expects 4 arguments: <host> <flag_id> <flag> <vuln>, got %d", len(args)-1)
		}
		vuln, err := strconv.Atoi(args[4])
		if err != nil {
			c.CheckFailedf("Checker failed", "invalid vuln number %q", args[4])
		}
		checker.Get(c, args[1], args[2], args[3], vuln)
	default:
		c.CheckFailedf("Checker failed", "bad action: %s", action)
	}
}

func writeResult(stdout, stderr io.Writer, res result) {
	if stdout != nil {
		_, _ = fmt.Fprintln(stdout, res.public)
	}
	if stderr != nil {
		_, _ = fmt.Fprintln(stderr, res.private)
	}
}
