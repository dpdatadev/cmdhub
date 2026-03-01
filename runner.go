package main

import (
	hub "dpdigital/cmdhub/api"
	"fmt"
)

func main() {
	//ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//defer cancel()
	fmt.Println("Hello, World!")
	/*c := execx.
	Command("printf", "hello\nworld\n").
	Pipe("tr", "a-z", "A-Z").
	Env("MODE=demo").
	WithContext(ctx).
	OnStdout(func(line string) {
		fmt.Println("OUT:", line)
	}).
	OnStderr(func(line string) {
		fmt.Println("ERR:", line)
	})
	*/
	cmd := hub.NewHubCommand("echo", []string{"Derek"}, "test")
	fmt.Println(cmd.ExecString())
	fmt.Println(cmd.GetUserName())
}
