/*
Sniperkit-Bot
- Date: 2018-08-11 22:28:44.32143264 +0200 CEST m=+0.117617904
- Status: analyzed
*/

package main

import (
	"strings"

	"github.com/fatih/color"
)

// Branch : A git branch
type Branch struct {
	Name    string
	Recency string
}

func (b *Branch) getDisplayString() string {
	return withPadding(b.Recency, 4) + coloredString(b.Name, b.getColor())
}

func (b *Branch) getColor() color.Attribute {
	switch b.getType() {
	case "feature":
		return color.FgGreen
	case "bugfix":
		return color.FgYellow
	case "hotfix":
		return color.FgRed
	default:
		return color.FgWhite
	}
}

// expected to return feature/bugfix/hotfix or blank string
func (b *Branch) getType() string {
	return strings.Split(b.Name, "/")[0]
}
