package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	junge "github.com/skipper-ad/junge-checkers"
	"github.com/skipper-ad/junge-checkers/require"
	o "github.com/skipper-ad/junge-checkers/require/options"
)

const port = 31337

func main() {
	junge.Main(junge.Handler{
		Config: junge.CheckerInfo{
			Vulns:      1,
			Timeout:    10,
			AttackData: true,
			Puts:       1,
			Gets:       1,
		},
		CheckFunc: check,
		PutFunc:   put,
		GetFunc:   get,
	})
}

func check(c *junge.C, host string) {
	reply := command(c, host, "PING")
	require.Equal(c, "PONG", reply, "Bad TCP protocol")
	c.OK("OK")
}

func put(c *junge.C, req junge.PutRequest) {
	c.Detail("vuln", req.Vuln)
	flagID := "flag-" + req.FlagID
	reply := command(c, req.Host, "PUT "+flagID+" "+req.Flag)
	require.Equal(c, "OK", reply, "Could not save flag", o.Corrupt())
	c.OK(flagID)
}

func get(c *junge.C, req junge.GetRequest) {
	c.Detail("flag_id", req.FlagID)
	reply := command(c, req.Host, "GET "+req.FlagID)
	require.Equal(c, req.Flag, reply, "Flag was corrupted", o.Corrupt())
	c.OK("OK")
}

func command(ctx context.Context, host, line string) string {
	c, ok := ctx.(*junge.C)
	if !ok {
		panic("tcp checker context must be *junge.C")
	}

	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		c.Down("Service is down", err.Error())
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if _, err := fmt.Fprintln(conn, line); err != nil {
		c.Down("Service is down", err.Error())
	}
	reply, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		c.Down("Service is down", err.Error())
	}
	return strings.TrimSpace(reply)
}
