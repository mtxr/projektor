package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/yamnikov-oleg/go-gtk/gio"
	"github.com/yamnikov-oleg/projektor/conf"
)

type EntryType int

const (
	ApplicatioEntry EntryType = iota
	CommandlineEntry
	FileEntry
	UrlEntry
	HistEntry
	CalcEntry
	WebSearchEntry
)

type LaunchEntry struct {
	Type EntryType
	// Clean name for an entry. E.g. "Atom Text Editor"
	Name string
	// Same as Name, but lowercased, e.g. "atom text editor"
	LoCaseName string
	// Formatted for display on a gtk widget, e.g. "<b>Ato</b>m Text Editor"
	MarkupName string
	// Name which is injected into search entry on Tab hit
	TabName string

	Icon string

	Cmdline string
	// Describes priority of an entry in results list. Lower index -> higher priority.
	QueryIndex int
}

func NewEntryFromDesktopFile(filepath string) (le *LaunchEntry, err error) {
	fd, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer fd.Close()

	cf, err := conf.Read(fd)
	if err != nil {
		return
	}

	section := cf.Sections["Desktop Entry"]
	if section.Bool("Hidden") {
		return nil, errors.New("desktop entry hidden")
	}
	if section.Bool("NoDisplay") {
		return nil, errors.New("desktop entry not displayed")
	}

	le = &LaunchEntry{Type: ApplicatioEntry}
	le.Name = section.Str("Name")
	le.LoCaseName = strings.ToLower(le.Name)
	le.Icon = section.Str("Icon")

	r := strings.NewReplacer(" %f", "", " %F", "", " %u", "", " %U", "")
	le.Cmdline = r.Replace(section.Str("Exec"))
	le.TabName = le.Cmdline

	return
}

func NewEntryFromCommand(command string) *LaunchEntry {
	return &LaunchEntry{
		Type:       CommandlineEntry,
		Icon:       "application-default-icon",
		Name:       command,
		MarkupName: "\u2192 <b>" + EscapeAmpersand(command) + "</b>",
		TabName:    command,
		Cmdline:    command,
	}
}

func NewEntryForFile(path string, displayName string, tabName string) (*LaunchEntry, error) {
	gFileInfo, fiErr := gio.NewFileForPath(path).QueryInfo("standard::*", gio.FILE_QUERY_INFO_NONE, nil)
	if fiErr != nil {
		return nil, fiErr
	}

	icon := gFileInfo.GetIcon()
	return &LaunchEntry{
		Type:       FileEntry,
		Icon:       icon.ToString(),
		Name:       path,
		MarkupName: EscapeAmpersand(displayName),
		TabName:    tabName,
		Cmdline:    "xdg-open " + path,
	}, nil
}

func NewUrlLaunchEntry(url string) *LaunchEntry {
	return &LaunchEntry{
		Type:       UrlEntry,
		Icon:       Config.URL.Icon,
		Name:       url,
		MarkupName: EscapeAmpersand(url),
		TabName:    url,
		Cmdline:    "xdg-open " + url,
	}
}

func NewCalcLaunchEntry(val float64) *LaunchEntry {
	valStr := fmt.Sprint(val)

	return &LaunchEntry{
		Type:       CalcEntry,
		Icon:       "gnome-calculator",
		Name:       valStr,
		MarkupName: fmt.Sprintf("= <b>%v</b>", val),
		TabName:    valStr,
		Cmdline:    "",
	}
}

func NewWebSearchEntry(q string) *LaunchEntry {
	return &LaunchEntry{
		Type:       WebSearchEntry,
		Icon:       Config.WebSearch.Icon,
		Name:       q,
		MarkupName: fmt.Sprintf("Search for: <b>%v</b>", EscapeHTML(q)),
		TabName:    q,
		Cmdline:    "xdg-open " + fmt.Sprintf(Config.WebSearch.Engine, url.QueryEscape(q)),
	}
}

func (le *LaunchEntry) UpdateMarkupName(index, length int) {
	index2 := index + length
	le.MarkupName = EscapeAmpersand(
		fmt.Sprintf("%v<b>%v</b>%v",
			le.Name[:index],
			le.Name[index:index2],
			le.Name[index2:],
		),
	)
	if le.Type == CommandlineEntry {
		le.MarkupName = "\u2192 " + le.MarkupName
	}
	if le.Type == HistEntry {
		le.MarkupName = "\u231a " + le.MarkupName
	}
}

type LaunchEntriesList []*LaunchEntry

func (list LaunchEntriesList) SortByName() {
	sort.Sort(lelSortableByName(list))
}

func (list LaunchEntriesList) SortByIndex() {
	sort.Sort(lelSortableByIndex(list))
}

type lelSortableByName LaunchEntriesList

func (list lelSortableByName) Len() int {
	return len(list)
}

func (list lelSortableByName) Less(i, j int) bool {
	return list[i].LoCaseName < list[j].LoCaseName
}

func (list lelSortableByName) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

type lelSortableByIndex LaunchEntriesList

func (list lelSortableByIndex) Len() int {
	return len(list)
}

func (list lelSortableByIndex) Less(i, j int) bool {
	if list[i].QueryIndex == list[j].QueryIndex {
		return list[i].LoCaseName < list[j].LoCaseName
	}
	return list[i].QueryIndex < list[j].QueryIndex
}

func (list lelSortableByIndex) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

type EntrySearchFunc func(string) LaunchEntriesList
