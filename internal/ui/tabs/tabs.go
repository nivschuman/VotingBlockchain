package tabs

import "fyne.io/fyne/v2"

type WidgetBuilder func() fyne.CanvasObject

type UiTab struct {
	Title         string
	WidgetBuilder WidgetBuilder
}

var MainTabs = []UiTab{
	{Title: "Blocks", WidgetBuilder: func() fyne.CanvasObject {
		return NewBlocksTab().widget
	}},
}
