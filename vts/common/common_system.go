package common

import "github.com/twitchylinux/ccr/vts"

var UserResourceClass = &vts.ResourceClass{
	Path: "common://resources/accounts:user",
	Name: "user",
}
var GroupResourceClass = &vts.ResourceClass{
	Path: "common://resources/accounts:group",
	Name: "group",
}
