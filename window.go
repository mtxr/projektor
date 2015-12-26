package main

import (
	"os"
	"os/exec"
	"strings"
	"unsafe"

	"github.com/yamnikov-oleg/go-gtk/gdk"
	"github.com/yamnikov-oleg/go-gtk/glib"
	"github.com/yamnikov-oleg/go-gtk/gtk"
)

var Ui struct {
	Window      *gtk.Window
	RootBox     *gtk.VBox
	SearchEntry *gtk.Entry
	ListStore   *gtk.ListStore
	ScrollWin   *gtk.ScrolledWindow
	TreeView    *gtk.TreeView
}

func setupSearchEntry() {
	Ui.SearchEntry = gtk.NewEntry()
	Ui.SearchEntry.GrabFocus()
}

func setupAppList() {
	Ui.TreeView = gtk.NewTreeView()
	Ui.TreeView.SetCanFocus(false)

	cr := gtk.NewCellRendererPixbuf()
	glib.ObjectFromNative(unsafe.Pointer(cr.ToCellRenderer())).Set("stock-size", int(gtk.ICON_SIZE_DIALOG))
	Ui.TreeView.AppendColumn(gtk.NewTreeViewColumnWithAttributes("Icon", cr, "icon-name", 0))
	Ui.TreeView.AppendColumn(gtk.NewTreeViewColumnWithAttributes("Id", gtk.NewCellRendererText(), "text", 1))

	Ui.ListStore = gtk.NewListStore(glib.G_TYPE_STRING, glib.G_TYPE_STRING, glib.G_TYPE_STRING)
	Ui.TreeView.SetModel(Ui.ListStore)

	Ui.ScrollWin = gtk.NewScrolledWindow(nil, nil)
	Ui.ScrollWin.SetCanFocus(false)
	Ui.ScrollWin.Add(Ui.TreeView)
}

func setupSearchLogic() {
	Ui.SearchEntry.Connect("changed", func() {

		Ui.ListStore.Clear()
		text := Ui.SearchEntry.GetText()
		text = strings.ToLower(text)
		if text == "" {
			return
		}
		reader := NewEntriesInterator()
		for reader.Next() {
			en := reader.Entry()
			if strings.Contains(en.LoCaseName, text) {
				listStoreAppendEntry(en)
			}
		}
		treeViewSelectFirst()

	})
}

func setupUiElements() {
	Ui.RootBox = gtk.NewVBox(false, 6)
	Ui.RootBox.SetCanFocus(false)

	setupSearchEntry()
	Ui.RootBox.PackStart(Ui.SearchEntry, false, false, 0)

	setupAppList()
	Ui.RootBox.PackEnd(Ui.ScrollWin, true, true, 0)

	setupSearchLogic()
}

func setupWindow() {
	Ui.Window = gtk.NewWindow(gtk.WINDOW_TOPLEVEL)

	Ui.Window.SetPosition(gtk.WIN_POS_CENTER)
	Ui.Window.SetGravity(gdk.GRAVITY_SOUTH)
	Ui.Window.SetDecorated(false)
	Ui.Window.SetSkipTaskbarHint(true)
	Ui.Window.SetBorderWidth(6)
	Ui.Window.SetSizeRequest(400, 300)

	Ui.Window.Connect("key-press-event", func(ctx *glib.CallbackContext) {
		arg := ctx.Args(0)
		e := *(**gdk.EventKey)(unsafe.Pointer(&arg))
		OnWindowKeyPress(e)
	})
	Ui.Window.Connect("focus-out-event", gtk.MainQuit)
	Ui.Window.Connect("destroy", gtk.MainQuit)

	setupUiElements()
	Ui.Window.Add(Ui.RootBox)
}

func StartUi() {
	gtk.Init(&os.Args)

	setupWindow()
	Ui.Window.ShowAll()

	gtk.Main()
}

func listStoreAppendEntry(entry *DtEntry) {
	var iter gtk.TreeIter
	Ui.ListStore.Append(&iter)
	Ui.ListStore.Set(&iter,
		0, entry.Icon,
		1, entry.Name,
		2, entry.Exec,
	)
}

func treeViewSelect(iter *gtk.TreeIter) {
	Ui.TreeView.GetSelection().SelectIter(iter)
	Ui.TreeView.ScrollToCell(Ui.ListStore.GetPath(iter), nil, false, 0, 0)
}

func treeViewSelectFirst() {
	var iter gtk.TreeIter
	if !Ui.ListStore.GetIterFirst(&iter) {
		return
	}
	treeViewSelect(&iter)
}

func OnWindowKeyPress(e *gdk.EventKey) {
	switch e.Keyval {

	case gdk.KEY_Escape:
		gtk.MainQuit()

	case gdk.KEY_Up, gdk.KEY_Down:
		selection := Ui.TreeView.GetSelection()
		if selection.CountSelectedRows() == 0 {
			return
		}
		var iter gtk.TreeIter
		selection.GetSelected(&iter)
		if e.Keyval == gdk.KEY_Up {
			if !Ui.ListStore.IterPrev(&iter) {
				return
			}
		} else {
			if !Ui.ListStore.IterNext(&iter) {
				return
			}
		}
		treeViewSelect(&iter)

	case gdk.KEY_Return:
		selection := Ui.TreeView.GetSelection()
		if selection.CountSelectedRows() == 0 {
			return
		}
		var iter gtk.TreeIter
		selection.GetSelected(&iter)
		var val glib.GValue
		Ui.ListStore.GetValue(&iter, 2, &val)
		cmd := strings.Fields(val.GetString())
		exec.Command(cmd[0], cmd[1:]...).Start()
		gtk.MainQuit()
	}
}
