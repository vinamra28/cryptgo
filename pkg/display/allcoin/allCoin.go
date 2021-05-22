package allcoin

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Gituser143/cryptgo/pkg/api"
	"github.com/Gituser143/cryptgo/pkg/display/coin"
	c "github.com/Gituser143/cryptgo/pkg/display/currency"
	"github.com/Gituser143/cryptgo/pkg/utils"
	"github.com/Gituser143/cryptgo/pkg/widgets"
	ui "github.com/gizak/termui/v3"
	"golang.org/x/sync/errgroup"
)

const (
	UP_ARROW   = "▲"
	DOWN_ARROW = "▼"
)

func DisplayAllCoins(ctx context.Context, dataChannel chan api.AssetData, sendData *bool) error {

	if err := ui.Init(); err != nil {
		return fmt.Errorf("failed to initialise termui: %v", err)
	}
	defer ui.Close()

	currency := "USD $"
	currencyVal := 1.0
	selectCurrency := false
	currencyWidget := c.NewCurrencyPage()

	coinSortIdx := -1
	coinSortAsc := false
	coinHeader := []string{
		"Rank",
		"Symbol",
		fmt.Sprintf("Price (%s)", currency),
		"Change %",
		"Supply / MaxSupply",
	}

	favSortIdx := -1
	favSortAsc := false
	favHeader := []string{
		"Symbol",
		fmt.Sprintf("Price (%s)", currency),
	}

	previousKey := ""

	coinIDs := make(map[string]string)

	myPage := NewAllCoinPage()
	selectedTable := myPage.CoinTable

	favourites := utils.GetFavourites()
	defer utils.SaveFavourites(favourites)

	help := widgets.NewHelpMenu()
	help.SelectHelpMenu("ALL")
	helpSelected := false

	pause := func() {
		*sendData = !(*sendData)
	}

	updateUI := func() {
		// Get Terminal Dimensions adn clear the UI
		w, h := ui.TerminalDimensions()
		myPage.Grid.SetRect(0, 0, w, h)

		ui.Clear()
		if helpSelected {
			help.Resize(w, h)
			ui.Render(help)
		} else if selectCurrency {
			currencyWidget.Resize(w, h)
			ui.Render(currencyWidget)
		} else {
			ui.Render(myPage.Grid)
		}
	}

	updateUI()

	uiEvents := ui.PollEvents()
	t := time.NewTicker(time.Duration(1) * time.Second)
	tick := t.C

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>": // q or Ctrl-C to quit
				return fmt.Errorf("UI Closed")

			case "<Resize>":
				updateUI()

			case "p":
				pause()

			case "?":
				helpSelected = !helpSelected
				updateUI()

			case "f":
				if !helpSelected {
					selectedTable.ShowCursor = false
					selectedTable = myPage.FavouritesTable
				}

			case "F":
				if !helpSelected {
					selectedTable.ShowCursor = false
					selectedTable = myPage.CoinTable
				}

			case "c":
				if !helpSelected {
					selectedTable.ShowCursor = false
					selectCurrency = true
					selectedTable.ShowCursor = true
					currencyWidget.UpdateRows()
					updateUI()
				}

			case "C":
				if !helpSelected {
					selectedTable.ShowCursor = false
					selectCurrency = true
					selectedTable.ShowCursor = true
					currencyWidget.UpdateAll()
					updateUI()
				}
			}
			if helpSelected {
				switch e.ID {
				case "?":
					updateUI()
				case "<Escape>":
					helpSelected = false
					updateUI()
				case "j", "<Down>":
					help.List.ScrollDown()
					ui.Render(help)
				case "k", "<Up>":
					help.List.ScrollUp()
					ui.Render(help)
				}
			} else if selectCurrency {
				switch e.ID {
				case "j", "<Down>":
					currencyWidget.ScrollDown()
				case "k", "<Up>":
					currencyWidget.ScrollUp()
				case "<C-d>":
					currencyWidget.ScrollHalfPageDown()
				case "<C-u>":
					currencyWidget.ScrollHalfPageUp()
				case "<C-f>":
					currencyWidget.ScrollPageDown()
				case "<C-b>":
					currencyWidget.ScrollPageUp()
				case "g":
					if previousKey == "g" {
						currencyWidget.ScrollTop()
					}
				case "<Home>":
					currencyWidget.ScrollTop()
				case "G", "<End>":
					currencyWidget.ScrollBottom()
				case "<Enter>":
					var err error
					if currencyWidget.SelectedRow < len(currencyWidget.Rows) {
						row := currencyWidget.Rows[currencyWidget.SelectedRow]
						currency = fmt.Sprintf("%s %s", row[0], row[1])
						currencyVal, err = strconv.ParseFloat(row[3], 64)
						if err != nil {
							currencyVal = 0
							currency = "USD $"
						}
						coinHeader[2] = fmt.Sprintf("Price (%s)", currency)
						favHeader[1] = fmt.Sprintf("Price (%s)", currency)
					}

					selectedTable = myPage.CoinTable
					selectCurrency = false

				case "<Escape>":
					selectedTable = myPage.CoinTable
					selectCurrency = false
				}
				if selectCurrency {
					ui.Render(currencyWidget)
				}
			} else if selectedTable != nil {
				selectedTable.ShowCursor = true

				switch e.ID {
				case "j", "<Down>":
					selectedTable.ScrollDown()
				case "k", "<Up>":
					selectedTable.ScrollUp()
				case "<C-d>":
					selectedTable.ScrollHalfPageDown()
				case "<C-u>":
					selectedTable.ScrollHalfPageUp()
				case "<C-f>":
					selectedTable.ScrollPageDown()
				case "<C-b>":
					selectedTable.ScrollPageUp()
				case "g":
					if previousKey == "g" {
						selectedTable.ScrollTop()
					}
				case "<Home>":
					selectedTable.ScrollTop()
				case "G", "<End>":
					selectedTable.ScrollBottom()

				case "s":
					id := ""
					symbol := ""

					// Get ID and symbol
					if selectedTable == myPage.CoinTable {
						if myPage.CoinTable.SelectedRow < len(myPage.CoinTable.Rows) {
							row := myPage.CoinTable.Rows[myPage.CoinTable.SelectedRow]
							symbol = row[1]
						}
					} else {
						if myPage.FavouritesTable.SelectedRow < len(myPage.FavouritesTable.Rows) {
							row := myPage.FavouritesTable.Rows[myPage.FavouritesTable.SelectedRow]
							symbol = row[0]
						}
					}
					id = coinIDs[symbol]

					favourites[id] = true

				case "S":
					id := ""
					symbol := ""

					// Get ID and symbol
					if selectedTable == myPage.CoinTable {
						if myPage.CoinTable.SelectedRow < len(myPage.CoinTable.Rows) {
							row := myPage.CoinTable.Rows[myPage.CoinTable.SelectedRow]
							symbol = row[1]
						}
					} else {
						if myPage.FavouritesTable.SelectedRow < len(myPage.FavouritesTable.Rows) {
							row := myPage.FavouritesTable.Rows[myPage.FavouritesTable.SelectedRow]
							symbol = row[0]
						}
					}
					id = coinIDs[symbol]

					delete(favourites, id)

				case "<Enter>":
					// pause UI and data send
					pause()

					id := ""
					symbol := ""

					// Get ID and symbol
					if selectedTable == myPage.CoinTable {
						if myPage.CoinTable.SelectedRow < len(myPage.CoinTable.Rows) {
							row := myPage.CoinTable.Rows[myPage.CoinTable.SelectedRow]
							symbol = row[1]
						}
					} else {
						if myPage.FavouritesTable.SelectedRow < len(myPage.FavouritesTable.Rows) {
							row := myPage.FavouritesTable.Rows[myPage.FavouritesTable.SelectedRow]
							symbol = row[0]
						}
					}
					id = coinIDs[symbol]

					if id != "" {
						eg, coinCtx := errgroup.WithContext(ctx)
						coinDataChannel := make(chan api.CoinData)
						coinPriceChannel := make(chan string)
						intervalChannel := make(chan string)

						ui.Clear()
						eg.Go(func() error {
							err := api.GetCoinHistory(
								coinCtx,
								id,
								intervalChannel,
								coinDataChannel,
							)
							return err
						})

						eg.Go(func() error {
							err := api.GetCoinAsset(coinCtx, id, coinDataChannel)
							return err
						})

						eg.Go(func() error {
							err := api.GetFavouritePrices(coinCtx,
								favourites,
								coinDataChannel,
							)
							return err
						})

						// Not run with eg because it blocks on ctx.Done()
						go api.GetLivePrice(coinCtx, id, coinPriceChannel)

						eg.Go(func() error {
							err := coin.DisplayCoin(
								coinCtx,
								id,
								intervalChannel,
								coinDataChannel,
								coinPriceChannel,
								uiEvents,
							)
							return err
						})

						if err := eg.Wait(); err != nil {
							if err.Error() != "UI Closed" {
								return err
							}
						}
						pause()
						updateUI()
					}
				}

				if selectedTable == myPage.CoinTable {
					switch e.ID {
					// Sort Ascending
					case "1", "2", "3", "4":
						idx, _ := strconv.Atoi(e.ID)
						coinSortIdx = idx - 1
						myPage.CoinTable.Header = append([]string{}, coinHeader...)
						myPage.CoinTable.Header[coinSortIdx] = coinHeader[coinSortIdx] + " " + UP_ARROW
						coinSortAsc = true
						utils.SortData(myPage.CoinTable.Rows, coinSortIdx, coinSortAsc, "COINS")

					// Sort Descending
					case "<F1>", "<F2>", "<F3>", "<F4>":
						myPage.CoinTable.Header = append([]string{}, coinHeader...)
						idx, _ := strconv.Atoi(e.ID[2:3])
						coinSortIdx = idx - 1
						myPage.CoinTable.Header[coinSortIdx] = coinHeader[coinSortIdx] + " " + DOWN_ARROW
						coinSortAsc = false
						utils.SortData(myPage.CoinTable.Rows, coinSortIdx, coinSortAsc, "COINS")

					}
				} else if selectedTable == myPage.FavouritesTable {
					switch e.ID {
					// Sort Ascending
					case "1", "2":
						idx, _ := strconv.Atoi(e.ID)
						favSortIdx = idx - 1
						myPage.FavouritesTable.Header = append([]string{}, favHeader...)
						myPage.FavouritesTable.Header[favSortIdx] = favHeader[favSortIdx] + " " + UP_ARROW
						favSortAsc = true
						utils.SortData(myPage.FavouritesTable.Rows, favSortIdx, favSortAsc, "FAVOURITES")

					// Sort Descending
					case "<F1>", "<F2>":
						myPage.FavouritesTable.Header = append([]string{}, favHeader...)
						idx, _ := strconv.Atoi(e.ID[2:3])
						favSortIdx = idx - 1
						myPage.FavouritesTable.Header[favSortIdx] = favHeader[favSortIdx] + " " + DOWN_ARROW
						favSortAsc = false
						utils.SortData(myPage.FavouritesTable.Rows, favSortIdx, favSortAsc, "FAVOURITES")
					}
				}

				ui.Render(myPage.Grid)
				if previousKey == "g" {
					previousKey = ""
				} else {
					previousKey = e.ID
				}
			}

		case data := <-dataChannel:
			if data.IsTopCoinData {
				for i, v := range data.TopCoinData {
					myPage.TopCoinGraphs[i].Title = " " + data.TopCoins[i] + " "
					myPage.TopCoinGraphs[i].Data["Value"] = v
					myPage.TopCoinGraphs[i].Labels["Value"] = fmt.Sprintf("%.2f %s", v[len(v)-1]/currencyVal, currency)
					myPage.TopCoinGraphs[i].Labels["Max"] = fmt.Sprintf("%.2f %s", utils.MaxFloat64(v...)/currencyVal, currency)
					myPage.TopCoinGraphs[i].Labels["Min"] = fmt.Sprintf("%.2f %s", utils.MinFloat64(v...)/currencyVal, currency)
				}
			} else {
				rows := [][]string{}
				favouritesData := [][]string{}
				myPage.CoinTable.Header[2] = fmt.Sprintf("Price (%s)", currency)
				myPage.FavouritesTable.Header[1] = fmt.Sprintf("Price (%s)", currency)
				for _, val := range data.Data {
					price := "NA"
					p, err := strconv.ParseFloat(val.PriceUsd, 64)
					if err == nil {
						price = fmt.Sprintf("%.2f", p/currencyVal)
					}

					change := "NA"
					c, err := strconv.ParseFloat(val.ChangePercent24Hr, 64)
					if err == nil {
						if c < 0 {
							change = fmt.Sprintf("%s %.2f", DOWN_ARROW, -1*c)
						} else {
							change = fmt.Sprintf("%s %.2f", UP_ARROW, c)
						}
					}

					s, err1 := strconv.ParseFloat(val.Supply, 64)
					ms, err2 := strconv.ParseFloat(val.MaxSupply, 64)

					units := ""
					var supplyVals []float64
					supplyData := ""

					if err1 == nil && err2 == nil {
						supplyVals, units = utils.RoundValues(s, ms)
						supplyData = fmt.Sprintf("%.2f%s / %.2f%s", supplyVals[0], units, supplyVals[1], units)
					} else {
						if err1 != nil {
							supplyVals, units = utils.RoundValues(s, ms)
							supplyData = fmt.Sprintf("NA / %.2f%s", supplyVals[1], units)
						} else {
							supplyVals, units = utils.RoundValues(s, ms)
							supplyData = fmt.Sprintf("%.2f%s / NA", supplyVals[0], units)
						}
					}

					rows = append(rows, []string{
						val.Rank,
						val.Symbol,
						price,
						change,
						supplyData,
					})

					if _, ok := coinIDs[val.Symbol]; !ok {
						coinIDs[val.Symbol] = val.Id
					}

					if _, ok := favourites[val.Id]; ok {
						favouritesData = append(favouritesData, []string{
							val.Symbol,
							price,
						})
					}
				}
				myPage.CoinTable.Rows = rows
				myPage.FavouritesTable.Rows = favouritesData

				if coinSortIdx != -1 {
					utils.SortData(myPage.CoinTable.Rows, coinSortIdx, coinSortAsc, "COINS")

					if coinSortAsc {
						myPage.CoinTable.Header[coinSortIdx] = coinHeader[coinSortIdx] + " " + UP_ARROW
					} else {
						myPage.CoinTable.Header[coinSortIdx] = coinHeader[coinSortIdx] + " " + DOWN_ARROW
					}
				}

				if favSortIdx != -1 {
					utils.SortData(myPage.FavouritesTable.Rows, favSortIdx, favSortAsc, "FAVOURITES")

					if favSortAsc {
						myPage.FavouritesTable.Header[favSortIdx] = favHeader[favSortIdx] + " " + UP_ARROW
					} else {
						myPage.FavouritesTable.Header[favSortIdx] = favHeader[favSortIdx] + " " + DOWN_ARROW
					}
				}
			}

		case <-tick:
			if *sendData {
				updateUI()
			}
		}

	}

}