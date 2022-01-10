package factory

import "github.com/spf13/cobra"

var subCommands = make(map[string]*cobra.Command)

func Register(name string, cmd *cobra.Command) {
	if _, ok := subCommands[name]; ok {
		panic("sub command already exists")
	}
	subCommands[name] = cmd
}

func Registered() []*cobra.Command {
	l := make([]*cobra.Command, len(subCommands))
	idx := 0
	for _, cmd := range subCommands {
		l[idx] = cmd
		idx++
	}
	return l
}
