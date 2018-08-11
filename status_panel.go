/*
Sniperkit-Bot
- Date: 2018-08-11 22:28:44.32143264 +0200 CEST m=+0.117617904
- Status: analyzed
*/

package main

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/jesseduffield/gocui"
)

func refreshStatus(g *gocui.Gui) error {
	v, err := g.View("status")
	if err != nil {
		panic(err)
	}
	// for some reason if this isn't wrapped in an update the clear seems to
	// be applied after the other things or something like that; the panel's
	// contents end up cleared
	g.Update(func(*gocui.Gui) error {
		v.Clear()
		pushables, pullables := gitUpstreamDifferenceCount()
		fmt.Fprint(v, "↑"+pushables+"↓"+pullables)
		branches := state.Branches
		if err := updateHasMergeConflictStatus(); err != nil {
			return err
		}
		if state.HasMergeConflicts {
			fmt.Fprint(v, coloredString(" (merging)", color.FgYellow))
		}

		if len(branches) == 0 {
			return nil
		}
		branch := branches[0]
		name := coloredString(branch.Name, branch.getColor())
		repo := getCurrentProject()
		fmt.Fprint(v, " "+repo+" → "+name)
		return nil
	})

	return nil
}
