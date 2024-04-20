package commandlib

import (
	"os"
	"sort"
)

type Command struct {
	Name   string
	Path   string
	Args   string
	Exists bool
}

func init() {
	sort.Slice(Commands, func(i, j int) bool {
		return Commands[i].Name < Commands[j].Name
	})

	for idx := range Commands {
		command := &Commands[idx]
		if _, err := os.Stat(command.Path); err == nil {
			(*command).Exists = true
		} else {
			(*command).Exists = false
		}
	}
}

func ExistingCommands() (ret []Command) {
	for _, command := range Commands {
		if command.Exists {
			ret = append(ret, command)
		}
	}
	return
}

func NonExistingCommands() (ret []Command) {
	for _, command := range Commands {
		if !command.Exists {
			ret = append(ret, command)
		}
	}
	return
}

var (
	Commands = []Command{
		{
			Name: "dockerps",
			Path: "/usr/bin/docker",
			Args: "ps -a -s",
		},
		{
			Name: "dockerimages",
			Path: "/usr/bin/docker",
			Args: "images",
		},
		{
			Name: "dockerstats",
			Path: "/usr/bin/docker",
			Args: "stats --no-stream --all",
		},
		{
			Name: "windockerps",
			Path: "C:\\Program Files\\Docker\\Docker\\resources\\bin\\docker.exe",
			Args: "ps -a -s",
		},
		{
			Name: "top",
			Path: "/usr/bin/top",
			Args: "-bn1 -w256",
		},
		{
			Name: "free",
			Path: "/usr/bin/free",
			Args: "-hw",
		},
		{
			Name: "df",
			Path: "/bin/df",
			Args: "-h",
		},
		{
			Name: "ipaddr",
			Path: "/usr/bin/ip",
			Args: "addr",
		},
		{
			Name: "iplink",
			Path: "/usr/bin/ip",
			Args: "link",
		},
		{
			Name: "netstat",
			Path: "/usr/bin/netstat",
			Args: "-taupen",
		},
		{
			Name: "sstu",
			Path: "/usr/bin/ss",
			Args: "-tu",
		},
		{
			Name: "sstul",
			Path: "/usr/bin/ss",
			Args: "-tul",
		},
		{
			Name: "pingv4",
			Path: "/bin/ping",
			Args: "-4 -c10 www.google.com",
		},
		{
			Name: "winpingv4",
			Path: "C:\\Windows\\System32\\PING.EXE",
			Args: "-4 www.google.com",
		},
		{
			Name: "pingv6",
			Path: "/bin/ping",
			Args: "-6 -c10 www.google.com",
		},
		{
			Name: "uname",
			Path: "/usr/bin/uname",
			Args: "-a",
		},
		{
			Name: "ps",
			Path: "/usr/bin/ps",
			Args: "auxf",
		},
		{
			Name: "uptime",
			Path: "/usr/bin/uptime",
			Args: "",
		},
		{
			Name: "systemctlstatus",
			Path: "/usr/bin/systemctl",
			Args: "status",
		},
		{
			Name: "smartctl",
			Path: "/usr/sbin/smartctl",
			Args: "/dev/nvme0 -a",
		},
	}
)
