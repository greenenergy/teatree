package teatree

import (
	"log"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// This is the material design icon in the nerdfont/material symbols set, as found in https://pictogrammers.com/library/mdi/
const ChevronRight = "\U000F0142"
const ChevronDown = "\U000F0140"
const Folder = "\U000F024B"
const File = "\U000F0214"

type ItemHolder interface {
	GetItems() []*TreeItem
	// GetPath - recursively search through the parent hierarchy and return the name of each item
	// They will be ordered from oldest ancestor to most recent descendant. The item itself will
	// be the last one in the list
	GetPath() []string
	AddChildren(...*TreeItem) ItemHolder
}

// Styling

var (
	unfocusedStyle = lipgloss.NewStyle().
			Border(lipgloss.HiddenBorder())
		//Background(lipgloss.Color("#000000"))
	focusedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))
	//Background(lipgloss.Color("#FFFFFF"))
)

type TreeItem struct {
	sync.Mutex
	Parent   ItemHolder
	Icon     string `json:"icon"`
	Name     string `json:"name"`
	Children []*TreeItem
	// CanHaveChildren: By setting this to True, you say that this item can have children. This allows for the implementation of a lazy loader, when you supply an Open() function. This affects how the item is rendered.
	CanHaveChildren bool
	Open            bool
	Data            interface{}
	OpenFunc        func(*TreeItem)
	CloseFunc       func(*TreeItem)
}

func (ti *TreeItem) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return ti, nil
}

func (ti *TreeItem) Init() tea.Cmd {
	return nil
}

func (ti *TreeItem) GetItems() []*TreeItem {
	return ti.Children
}

func (ti *TreeItem) View() string {
	// Return the view string for myself plus my children if I am open
	//return " " + "󰅂" + ti.Icon + " " + ti.Name
	s := ti.Icon + " " + ti.Name
	if len(ti.Children) > 0 && ti.Open {
		var kids []string
		for _, item := range ti.Children {
			kids = append(kids, item.View())
		}
		s += "\n" + strings.Join(kids, "\n")
	}
	return s
}

func (ti *TreeItem) GetPath() []string {
	var path []string
	if ti.Parent != nil {
		path = ti.Parent.GetPath()
	}
	return append(path, ti.Name)
}

func (ti *TreeItem) OpenChildren() {
	ti.Open = true
}

func (ti *TreeItem) CloseChildren() {
	ti.Open = false
}

func (ti *TreeItem) ToggleChildren() {
	if ti.CanHaveChildren {
		ti.Open = !ti.Open
		if ti.Open {
			if ti.OpenFunc != nil {
				ti.OpenFunc(ti)
			}
		} else {
			if ti.CloseFunc != nil {
				ti.CloseFunc(ti)
			}
		}
	}
}

// AddChildren - adds a list of children to an item, and then returns the item
// AddChild - adds a child item to the item. Adding a child will result in the automatic inclusion of
// the collapse chevron
func (ti *TreeItem) AddChildren(children ...*TreeItem) ItemHolder {
	// TODO: Should this do any mutex
	ti.Lock()
	ti.Children = append(ti.Children, children...)
	ti.Unlock()
	ti.CanHaveChildren = true // If it wasn't set before, it will be now

	for _, child := range children {
		child.Parent = ti
	}

	return ti
}

func NewItem(name, icon string, canHaveChildren bool, children []*TreeItem, openFunc, closeFunc func(*TreeItem), data interface{}) *TreeItem {
	return &TreeItem{
		Name:            name,
		Icon:            icon,
		Children:        children,
		Open:            false,
		OpenFunc:        openFunc,
		CloseFunc:       closeFunc,
		Data:            data,
		CanHaveChildren: canHaveChildren,
	}
}

type KeyMap struct {
	Space    key.Binding
	GoToTop  key.Binding
	GoToLast key.Binding
	Down     key.Binding
	Up       key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Back     key.Binding
	Open     key.Binding
	Select   key.Binding
}

type Tree struct {
	sync.Mutex
	TopLine              int // for scrolling
	Width                int
	Height               int
	ClosedChildrenSymbol string
	OpenChildrenSymbol   string
	ActiveItem           *TreeItem
	Items                []*TreeItem
	initialized          bool
	Style                lipgloss.Style
	KeyMap               KeyMap
	quitting             bool
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Space:    key.NewBinding(key.WithKeys(" "), key.WithHelp(" ", "space")),
		GoToTop:  key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "first")),
		GoToLast: key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "last")),
		Down:     key.NewBinding(key.WithKeys("j", "down", "ctrl+n"), key.WithHelp("j", "down")),
		Up:       key.NewBinding(key.WithKeys("k", "up", "ctrl+p"), key.WithHelp("k", "up")),
		PageUp:   key.NewBinding(key.WithKeys("K", "pgup"), key.WithHelp("pgup", "page up")),
		PageDown: key.NewBinding(key.WithKeys("J", "pgdown"), key.WithHelp("pgdown", "page down")),
		Back:     key.NewBinding(key.WithKeys("h", "backspace", "left", "esc"), key.WithHelp("h", "back")),
		Open:     key.NewBinding(key.WithKeys("l", "right", "enter"), key.WithHelp("l", "open")),
		Select:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	}
}

func New() tea.Model {
	t := Tree{
		OpenChildrenSymbol:   ChevronDown,
		ClosedChildrenSymbol: ChevronRight,
		KeyMap:               DefaultKeyMap(),
	}
	t.setInitialValues()
	return &t
}

func (t *Tree) setInitialValues() {
	t.initialized = true
}

func (t *Tree) AddChildren(i ...*TreeItem) ItemHolder {
	if len(i) == 0 {
		return t
	}
	t.Lock()
	t.Items = append(t.Items, i...)
	t.Unlock()
	// After we add the items, if we didn't have an active item, let's make it the first
	// one in the list
	if t.ActiveItem == nil {
		t.ActiveItem = t.Items[0]
	}
	for _, item := range i {
		item.Parent = t
	}
	return t
}

func (t *Tree) GetItems() []*TreeItem {
	return t.Items
}

func (t *Tree) GetPath() []string {
	return []string{}
}

func (t *Tree) Init() tea.Cmd {
	return nil
}

func (t *Tree) Render() {
	log.Println("At Tree.Render()")
}

// SelectPrevious - selects the previous TreeItem. This involves first getting the parent and then telling the parent to select the previous item from the current selection. If we're already at the first child, then we go to the grandparent and select the previous parent item from us, and then we descend to the most open child and activate that.rune
func (t *Tree) SelectPrevious() {
	active := t.ActiveItem
	parent := active.Parent
	parentItems := parent.GetItems()

	for x, item := range parentItems {
		if item == active {
			if x-1 >= 0 {
				t.ActiveItem = parentItems[x-1]
			}
			return
		}
	}
	log.Println("prev not found")
}

// SelectNext is like SelectPrevious, but the other way
func (t *Tree) SelectNext() {
	active := t.ActiveItem
	parent := active.Parent
	parentItems := parent.GetItems()

	for x, item := range parentItems {
		if item == active {
			if x+1 < len(parentItems) {
				t.ActiveItem = parentItems[x+1]
			}
			return
		}
	}
	log.Println("next not found")
}

// ToggleChild will toggle the open/closed state of the current selection. This only has meaning if there
// are actually children
func (t *Tree) ToggleChild() {
	if t.ActiveItem != nil {
		t.ActiveItem.ToggleChildren()
	} else {
		log.Println("No active items to toggle")
	}
}

func (t *Tree) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf("tree update, msg type: %T\n", msg)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// TODO: Do I take into account margin & border?
		t.Width = msg.Width
		t.Height = msg.Height
		t.initialized = true

	// TODO: Convert these simple strings to a configurable keymap
	case tea.KeyMsg:
		switch msg.String() {
		case "?":
			log.Println("info")
		case "up", "k":
			t.SelectPrevious()
		case "down", "j":
			t.SelectNext()
		case "ctrl+c", "q":
			t.quitting = true
			return t, tea.Quit
		case " ", ".":
			t.ToggleChild()
			return t, nil
		}
	}

	var cmd tea.Cmd
	if t.ActiveItem != nil {
		var i tea.Model
		i, cmd = t.ActiveItem.Update(msg)
		t.ActiveItem = i.(*TreeItem)
	}
	return t, cmd
}

func (t *Tree) View() string {
	if !t.initialized {
		return ""
	}
	var views []string
	// Iterate through the children, calling View() on each of them.
	for _, item := range t.Items {
		s := " "
		if item.CanHaveChildren {
			if item.Open {
				s = ChevronDown
			} else {
				s = ChevronRight
			}
		}
		s += item.View()
		if item == t.ActiveItem {
			views = append(views, focusedStyle.Render(s))
		} else {
			views = append(views, unfocusedStyle.Render(s))
		}
	}
	//return strings.Join(views, "\n")

	return lipgloss.JoinVertical(
		lipgloss.Left, views...,
	)

}
