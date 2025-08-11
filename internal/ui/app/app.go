package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"github.com/nivschuman/VotingBlockchain/internal/database/repositories"
	"github.com/nivschuman/VotingBlockchain/internal/ui/tabs"
	"gorm.io/gorm"
)

type AppBuilder interface {
	BuildApp() App
}

type AppBuilderImpl struct {
	blockRepository       repositories.BlockRepository
	transactionRepository repositories.TransactionRepository
}

type App interface {
	Start()
}

type AppImpl struct {
	fyneApp    fyne.App
	mainWindow fyne.Window
}

func NewAppBuilderImpl(db *gorm.DB) *AppBuilderImpl {
	transactionRepository := repositories.NewTransactionRepositoryImpl(db)
	blockRepository := repositories.NewBlockRepositoryImpl(db, transactionRepository)

	return &AppBuilderImpl{
		transactionRepository: transactionRepository,
		blockRepository:       blockRepository,
	}
}

func (appBuilder *AppBuilderImpl) BuildApp() App {
	a := app.New()
	w := a.NewWindow("Blockchain UI")

	t := container.NewAppTabs()

	blocksTab := tabs.NewBlocksTab(appBuilder.blockRepository)
	t.Append(container.NewTabItem("Blocks", blocksTab.GetWidget()))

	w.SetContent(t)
	w.Resize(fyne.NewSize(800, 600))
	return &AppImpl{fyneApp: a, mainWindow: w}
}

func (app *AppImpl) Start() {
	app.mainWindow.ShowAndRun()
}
