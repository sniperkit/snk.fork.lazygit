/*
Sniperkit-Bot
- Date: 2018-08-11 22:28:44.32143264 +0200 CEST m=+0.117617904
- Status: analyzed
*/

package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jesseduffield/gocui"
)

var cyclableViews = []string{"files", "branches", "commits", "stash"}

func refreshSidePanels(g *gocui.Gui) error {
	refreshBranches(g)
	refreshFiles(g)
	refreshCommits(g)
	return nil
}

func nextView(g *gocui.Gui, v *gocui.View) error {
	var focusedViewName string
	if v == nil || v.Name() == cyclableViews[len(cyclableViews)-1] {
		focusedViewName = cyclableViews[0]
	} else {
		for i := range cyclableViews {
			if v.Name() == cyclableViews[i] {
				focusedViewName = cyclableViews[i+1]
				break
			}
			if i == len(cyclableViews)-1 {
				devLog(v.Name() + " is not in the list of views")
				return nil
			}
		}
	}
	focusedView, err := g.View(focusedViewName)
	if err != nil {
		panic(err)
	}
	return switchFocus(g, v, focusedView)
}

func previousView(g *gocui.Gui, v *gocui.View) error {
	var focusedViewName string
	if v == nil || v.Name() == cyclableViews[0] {
		focusedViewName = cyclableViews[len(cyclableViews)-1]
	} else {
		for i := range cyclableViews {
			if v.Name() == cyclableViews[i] {
				focusedViewName = cyclableViews[i-1] // TODO: make this work properly
				break
			}
			if i == len(cyclableViews)-1 {
				devLog(v.Name() + " is not in the list of views")
				return nil
			}
		}
	}
	focusedView, err := g.View(focusedViewName)
	if err != nil {
		panic(err)
	}
	return switchFocus(g, v, focusedView)
}

func newLineFocused(g *gocui.Gui, v *gocui.View) error {
	mainView, _ := g.View("main")
	mainView.SetOrigin(0, 0)

	switch v.Name() {
	case "files":
		return handleFileSelect(g, v)
	case "branches":
		return handleBranchSelect(g, v)
	case "confirmation":
		return nil
	case "commitMessage":
		return handleCommitFocused(g, v)
	case "main":
		// TODO: pull this out into a 'view focused' function
		refreshMergePanel(g)
		v.Highlight = false
		return nil
	case "commits":
		return handleCommitSelect(g, v)
	case "stash":
		return handleStashEntrySelect(g, v)
	default:
		panic("No view matching newLineFocused switch statement")
	}
}

func returnFocus(g *gocui.Gui, v *gocui.View) error {
	previousView, err := g.View(state.PreviousView)
	if err != nil {
		panic(err)
	}
	return switchFocus(g, v, previousView)
}

// pass in oldView = nil if you don't want to be able to return to your old view
func switchFocus(g *gocui.Gui, oldView, newView *gocui.View) error {
	// we assume we'll never want to return focus to a confirmation panel i.e.
	// we should never stack confirmation panels
	if oldView != nil && oldView.Name() != "confirmation" {
		oldView.Highlight = false
		devLog("setting previous view to:", oldView.Name())
		state.PreviousView = oldView.Name()
	}
	newView.Highlight = true
	devLog("new focused view is " + newView.Name())
	if _, err := g.SetCurrentView(newView.Name()); err != nil {
		return err
	}
	g.Cursor = newView.Editable
	return newLineFocused(g, newView)
}

func getItemPosition(v *gocui.View) int {
	_, cy := v.Cursor()
	_, oy := v.Origin()
	return oy + cy
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	// swallowing cursor movements in main
	// TODO: pull this out
	if v == nil || v.Name() == "main" {
		return nil
	}

	ox, oy := v.Origin()
	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
		if err := v.SetOrigin(ox, oy-1); err != nil {
			return err
		}
	}

	newLineFocused(g, v)
	return nil
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	// swallowing cursor movements in main
	// TODO: pull this out
	if v == nil || v.Name() == "main" {
		return nil
	}
	cx, cy := v.Cursor()
	ox, oy := v.Origin()
	if cy+oy >= len(v.BufferLines())-2 {
		return nil
	}
	if err := v.SetCursor(cx, cy+1); err != nil {
		if err := v.SetOrigin(ox, oy+1); err != nil {
			return err
		}
	}

	newLineFocused(g, v)
	return nil
}

func resetOrigin(v *gocui.View) error {
	if err := v.SetCursor(0, 0); err != nil {
		return err
	}
	return v.SetOrigin(0, 0)
}

// if the cursor down past the last item, move it up one
func correctCursor(v *gocui.View) error {
	cx, cy := v.Cursor()
	_, oy := v.Origin()
	lineCount := len(v.BufferLines()) - 2
	if cy >= lineCount-oy {
		return v.SetCursor(cx, lineCount-oy)
	}
	return nil
}

func renderString(g *gocui.Gui, viewName, s string) error {
	g.Update(func(*gocui.Gui) error {
		v, err := g.View(viewName)
		// just in case the view disappeared as this function was called, we'll
		// silently return if it's not found
		if err != nil {
			return nil
		}
		v.Clear()
		fmt.Fprint(v, s)
		v.Wrap = true
		return nil
	})
	return nil
}

func optionsMapToString(optionsMap map[string]string) string {
	optionsArray := make([]string, 0)
	for key, description := range optionsMap {
		optionsArray = append(optionsArray, key+": "+description)
	}
	sort.Strings(optionsArray)
	return strings.Join(optionsArray, ", ")
}

func renderOptionsMap(g *gocui.Gui, optionsMap map[string]string) error {
	return renderString(g, "options", optionsMapToString(optionsMap))
}

func loader() string {
	characters := "|/-\\"
	now := time.Now()
	nanos := now.UnixNano()
	index := nanos / 50000000 % int64(len(characters))
	return characters[index : index+1]
}

// TODO: refactor properly
func getFilesView(g *gocui.Gui) *gocui.View {
	v, _ := g.View("files")
	return v
}

func getCommitsView(g *gocui.Gui) *gocui.View {
	v, _ := g.View("commits")
	return v
}

func getCommitMessageView(g *gocui.Gui) *gocui.View {
	v, _ := g.View("commitMessage")
	return v
}
