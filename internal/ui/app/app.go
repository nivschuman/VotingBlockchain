package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"github.com/nivschuman/VotingBlockchain/internal/ui/tabs"
)

type App struct {
	fyneApp    fyne.App
	mainWindow fyne.Window
}

func MainApp() *App {
	a := app.New()
	w := a.NewWindow("Blockchain UI")

	t := container.NewAppTabs()
	for _, tab := range tabs.MainTabs {
		t.Append(container.NewTabItem(tab.Title, tab.WidgetBuilder()))
	}

	w.SetContent(t)
	w.Resize(fyne.NewSize(800, 600))
	return &App{fyneApp: a, mainWindow: w}
}

func (app *App) Start() {
	app.mainWindow.ShowAndRun()
}
