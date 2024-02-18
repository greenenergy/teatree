package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/greenenergy/teatree"
)

const IconFolder = "\U000F024B"
const IconFile = "\U000F0214"

var (
	folderColor = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFCF00")) // yellow
	fileColor = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FFFF")) // cyan
)

type FileBrowserModel struct {
	dir  string
	Tree *teatree.Tree
}

func (fm *FileBrowserModel) Init() tea.Cmd {
	return nil
}

func (fm *FileBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return fm.Tree.Update(msg)
}

func (fm *FileBrowserModel) View() string {
	return fm.Tree.View()
}

func (fm *FileBrowserModel) walk(p string, item teatree.ItemHolder) error {
	err := filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		var icon string
		canHaveChildren := d.IsDir()

		if d.IsDir() {
			icon = folderColor.Render(IconFolder)
		} else {
			icon = fileColor.Render(IconFile)
		}

		openFunc := func(ti *teatree.TreeItem) {
			// This function is called when the user toggles an item that can have children. For now that only means this is a folder and we are now supposed to walk the ti's path, adding items
			// If we have no children, then we should walk the directroy.
			// If we DO have children, there should be some way to bust the cache
			if len(ti.Children) == 0 {
				fullpath := strings.Join(ti.GetPath(), "/")
				//fmt.Println("supposed to walk:", fullpath)
				err = fm.walk(fullpath, ti)
			}
		}
		var children []*teatree.TreeItem
		item := teatree.NewItem(path, icon, canHaveChildren, children, openFunc, nil, nil)
		// TODO: Need to add OpenFunc() that can walk another hierarchy level.
		fm.Tree.AddChildren(item)

		if d.IsDir() && path != p {
			// Do not descend into subdirectories
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		fmt.Printf("error walking the path %q: %v\n", p, err)
		return err
	}
	return nil
}

func New(dir string) tea.Model {
	fm := &FileBrowserModel{
		dir:  dir,
		Tree: teatree.New().(*teatree.Tree),
	}
	if err := fm.walk(dir, fm.Tree); err != nil {
		log.Fatal(err)
	}
	return fm
}

func main() {
	//dir := "/home/cfox"
	dir := "."
	m := New(dir)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
