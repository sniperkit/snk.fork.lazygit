/*
Sniperkit-Bot
- Date: 2018-08-11 22:28:44.32143264 +0200 CEST m=+0.117617904
- Status: analyzed
*/

package main

import (
	// "io"
	// "io/ioutil"
	// "strings"
	"errors"
	"strings"

	"github.com/fatih/color"
	"github.com/jesseduffield/gocui"
)

var (
	errNoFiles    = errors.New("No changed files")
	errNoUsername = errors.New(`No username set. Please do: git config --global user.name "Your Name"`)
)

func stagedFiles(files []GitFile) []GitFile {
	result := make([]GitFile, 0)
	for _, file := range files {
		if file.HasStagedChanges {
			result = append(result, file)
		}
	}
	return result
}

func stageSelectedFile(g *gocui.Gui) error {
	file, err := getSelectedFile(g)
	if err != nil {
		return err
	}
	return stageFile(file.Name)
}

func handleFilePress(g *gocui.Gui, v *gocui.View) error {
	file, err := getSelectedFile(g)
	if err != nil {
		if err == errNoFiles {
			return nil
		}
		return err
	}

	if file.HasMergeConflicts {
		return handleSwitchToMerge(g, v)
	}

	if file.HasUnstagedChanges {
		stageFile(file.Name)
	} else {
		unStageFile(file.Name, file.Tracked)
	}

	if err := refreshFiles(g); err != nil {
		return err
	}

	return handleFileSelect(g, v)
}

func handleAddPatch(g *gocui.Gui, v *gocui.View) error {
	file, err := getSelectedFile(g)
	if err != nil {
		if err == errNoFiles {
			return nil
		}
		return err
	}
	if !file.HasUnstagedChanges {
		return createErrorPanel(g, "File has no unstaged changes to add")
	}
	if !file.Tracked {
		return createErrorPanel(g, "Cannot git add --patch untracked files")
	}
	gitAddPatch(g, file.Name)
	return err
}

func getSelectedFile(g *gocui.Gui) (GitFile, error) {
	if len(state.GitFiles) == 0 {
		return GitFile{}, errNoFiles
	}
	filesView, err := g.View("files")
	if err != nil {
		panic(err)
	}
	lineNumber := getItemPosition(filesView)
	return state.GitFiles[lineNumber], nil
}

func handleFileRemove(g *gocui.Gui, v *gocui.View) error {
	file, err := getSelectedFile(g)
	if err != nil {
		if err == errNoFiles {
			return nil
		}
		return err
	}
	var deleteVerb string
	if file.Tracked {
		deleteVerb = "checkout"
	} else {
		deleteVerb = "delete"
	}
	return createConfirmationPanel(g, v, strings.Title(deleteVerb)+" file", "Are you sure you want to "+deleteVerb+" "+file.Name+" (you will lose your changes)?", func(g *gocui.Gui, v *gocui.View) error {
		if err := removeFile(file); err != nil {
			panic(err)
		}
		return refreshFiles(g)
	}, nil)
}

func handleIgnoreFile(g *gocui.Gui, v *gocui.View) error {
	file, err := getSelectedFile(g)
	if err != nil {
		return createErrorPanel(g, err.Error())
	}
	if file.Tracked {
		return createErrorPanel(g, "Cannot ignore tracked files")
	}
	gitIgnore(file.Name)
	return refreshFiles(g)
}

func renderfilesOptions(g *gocui.Gui, gitFile *GitFile) error {
	optionsMap := map[string]string{
		"← → ↑ ↓":   "navigate",
		"S":         "stash files",
		"c":         "commit changes",
		"o":         "open",
		"i":         "ignore",
		"d":         "delete",
		"space":     "toggle staged",
		"R":         "refresh",
		"t":         "add patch",
		"e":         "edit",
		"PgUp/PgDn": "scroll",
	}
	if state.HasMergeConflicts {
		optionsMap["a"] = "abort merge"
		optionsMap["m"] = "resolve merge conflicts"
	}
	if gitFile == nil {
		return renderOptionsMap(g, optionsMap)
	}
	if gitFile.Tracked {
		optionsMap["d"] = "checkout"
	}
	return renderOptionsMap(g, optionsMap)
}

func handleFileSelect(g *gocui.Gui, v *gocui.View) error {
	gitFile, err := getSelectedFile(g)
	if err != nil {
		if err != errNoFiles {
			return err
		}
		renderString(g, "main", "No changed files")
		return renderfilesOptions(g, nil)
	}
	renderfilesOptions(g, &gitFile)
	var content string
	if gitFile.HasMergeConflicts {
		return refreshMergePanel(g)
	}

	content = getDiff(gitFile)
	return renderString(g, "main", content)
}

func handleCommitPress(g *gocui.Gui, filesView *gocui.View) error {
	if len(stagedFiles(state.GitFiles)) == 0 && !state.HasMergeConflicts {
		return createErrorPanel(g, "There are no staged files to commit")
	}
	commitMessageView := getCommitMessageView(g)
	g.Update(func(g *gocui.Gui) error {
		g.SetViewOnTop("commitMessage")
		switchFocus(g, filesView, commitMessageView)
		return nil
	})
	return nil
}

func handleCommitEditorPress(g *gocui.Gui, filesView *gocui.View) error {
	if len(stagedFiles(state.GitFiles)) == 0 && !state.HasMergeConflicts {
		return createErrorPanel(g, "There are no staged files to commit")
	}
	runSubProcess(g, "git", "commit")
	return nil
}

func genericFileOpen(g *gocui.Gui, v *gocui.View, open func(*gocui.Gui, string) (string, error)) error {
	file, err := getSelectedFile(g)
	if err != nil {
		if err != errNoFiles {
			return err
		}
		return nil
	}
	if _, err := open(g, file.Name); err != nil {
		return createErrorPanel(g, err.Error())
	}
	return nil
}

func handleFileEdit(g *gocui.Gui, v *gocui.View) error {
	return genericFileOpen(g, v, editFile)
}

func handleFileOpen(g *gocui.Gui, v *gocui.View) error {
	return genericFileOpen(g, v, openFile)
}

func handleSublimeFileOpen(g *gocui.Gui, v *gocui.View) error {
	return genericFileOpen(g, v, sublimeOpenFile)
}

func handleVsCodeFileOpen(g *gocui.Gui, v *gocui.View) error {
	return genericFileOpen(g, v, vsCodeOpenFile)
}

func handleRefreshFiles(g *gocui.Gui, v *gocui.View) error {
	return refreshFiles(g)
}

func refreshStateGitFiles() {
	// get files to stage
	gitFiles := getGitStatusFiles()
	state.GitFiles = mergeGitStatusFiles(state.GitFiles, gitFiles)
	updateHasMergeConflictStatus()
}

func updateHasMergeConflictStatus() error {
	merging, err := isInMergeState()
	if err != nil {
		return err
	}
	state.HasMergeConflicts = merging
	return nil
}

func renderGitFile(gitFile GitFile, filesView *gocui.View) {
	// potentially inefficient to be instantiating these color
	// objects with each render
	red := color.New(color.FgRed)
	green := color.New(color.FgGreen)
	if !gitFile.Tracked && !gitFile.HasStagedChanges {
		red.Fprintln(filesView, gitFile.DisplayString)
		return
	}
	green.Fprint(filesView, gitFile.DisplayString[0:1])
	red.Fprint(filesView, gitFile.DisplayString[1:3])
	if gitFile.HasUnstagedChanges {
		red.Fprintln(filesView, gitFile.Name)
	} else {
		green.Fprintln(filesView, gitFile.Name)
	}
}

func catSelectedFile(g *gocui.Gui) (string, error) {
	item, err := getSelectedFile(g)
	if err != nil {
		if err != errNoFiles {
			return "", err
		}
		return "", renderString(g, "main", "No file to display")
	}
	cat, err := catFile(item.Name)
	if err != nil {
		panic(err)
	}
	return cat, nil
}

func refreshFiles(g *gocui.Gui) error {
	filesView, err := g.View("files")
	if err != nil {
		return err
	}
	refreshStateGitFiles()
	filesView.Clear()
	for _, gitFile := range state.GitFiles {
		renderGitFile(gitFile, filesView)
	}
	correctCursor(filesView)
	if filesView == g.CurrentView() {
		handleFileSelect(g, filesView)
	}
	return nil
}

func pullFiles(g *gocui.Gui, v *gocui.View) error {
	createMessagePanel(g, v, "", "Pulling...")
	go func() {
		if output, err := gitPull(); err != nil {
			createErrorPanel(g, output)
		} else {
			closeConfirmationPrompt(g)
			refreshCommits(g)
			refreshStatus(g)
		}
		refreshFiles(g)
	}()
	return nil
}

func pushFiles(g *gocui.Gui, v *gocui.View) error {
	createMessagePanel(g, v, "", "Pushing...")
	go func() {
		if output, err := gitPush(); err != nil {
			createErrorPanel(g, output)
		} else {
			closeConfirmationPrompt(g)
			refreshCommits(g)
			refreshStatus(g)
		}
	}()
	return nil
}

func handleSwitchToMerge(g *gocui.Gui, v *gocui.View) error {
	mergeView, err := g.View("main")
	if err != nil {
		return err
	}
	file, err := getSelectedFile(g)
	if err != nil {
		if err != errNoFiles {
			return err
		}
		return nil
	}
	if !file.HasMergeConflicts {
		return createErrorPanel(g, "This file has no merge conflicts")
	}
	switchFocus(g, v, mergeView)
	return refreshMergePanel(g)
}

func handleAbortMerge(g *gocui.Gui, v *gocui.View) error {
	output, err := gitAbortMerge()
	if err != nil {
		return createErrorPanel(g, output)
	}
	createMessagePanel(g, v, "", "Merge aborted")
	refreshStatus(g)
	return refreshFiles(g)
}

func handleResetHard(g *gocui.Gui, v *gocui.View) error {
	return createConfirmationPanel(g, v, "Clear file panel", "Are you sure you want `reset --hard HEAD`? You may lose changes", func(g *gocui.Gui, v *gocui.View) error {
		if err := gitResetHard(); err != nil {
			createErrorPanel(g, err.Error())
		}
		return refreshFiles(g)
	}, nil)
}
