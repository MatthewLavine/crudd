package commandlib

type Command struct {
	Name string
	Path string
	Args string
}

var (
	Commands = []Command{
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
			Name: "iplink",
			Path: "/usr/bin/ip",
			Args: "link",
		},
		{
			Name: "pingv4",
			Path: "/bin/ping",
			Args: "-4 -c3 www.google.com",
		},
		{
			Name: "pingv6",
			Path: "/bin/ping",
			Args: "-6 -c3 www.google.com",
		},
		{
			Name: "uname",
			Path: "/usr/bin/uname",
			Args: "-a",
		},
		{
			Name: "systemctlstatus",
			Path: "/usr/bin/systemctl",
			Args: "status",
		},
	}
)
