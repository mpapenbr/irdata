//nolint:funlen,lll // tmp file
package splits

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"

	"github.com/mpapenbr/irdata/cmd/util"
	"github.com/mpapenbr/irdata/irdata"
	"github.com/mpapenbr/irdata/log"
)

func NewSplitsDataCommand() *cobra.Command {
	subsessionID := 0
	carIDs := []int{}
	carClassIDs := []int{}
	splitCond := ""
	components := []string{}
	showIDs := false
	doPlots := false
	cmd := cobra.Command{
		Use:   "data",
		Short: "collect split data from iRacing",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			runner := collectSplitsDataCommand{
				subsessionID: subsessionID,
				carIDs:       carIDs,
				carClassIDs:  carClassIDs,
				splitCond:    splitCond,
				components:   components,
				showIDs:      showIDs,
				plot:         doPlots,
			}
			if err := runner.run(cmd.Context()); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().
		IntVar(&subsessionID, "subsession-id", 0, "iRacing subsession ID to collect splits data for")
	cmd.Flags().IntSliceVar(&carIDs, "car-id", []int{}, "iRacing car ID to collect splits data for")
	cmd.Flags().
		IntSliceVar(&carClassIDs, "car-class-id", []int{}, "iRacing car class ID to collect splits data for")
	cmd.Flags().BoolVar(&showIDs, "show-ids", false, "whether to include IDs in the output")
	cmd.Flags().BoolVar(&doPlots, "plot", false, "whether to generate plots for the splits data")
	cmd.Flags().StringVar(&splitCond, "splits", "", "Condition to filter splits data")
	cmd.Flags().StringSliceVar(&components, "component",
		[]string{"summary", "class", "car"},
		"Components to include in the output")
	return &cmd
}

type (
	teamData struct {
		split         int
		carID         int
		carClassID    int
		num           int
		rating        int
		carNumber     string
		driverRatings []int
	}
	minMaxAvg struct {
		min int
		max int
		avg int
		num int
	}
	statData struct {
		id                   int
		name                 string
		numTeams, numDrivers int
		ratings              *minMaxAvg
		driverRatings        *minMaxAvg
	}

	collectSplitsDataCommand struct {
		subsessionID         int
		carIDs               []int
		carClassIDs          []int
		splitCond            string
		components           []string
		showIDs              bool
		plot                 bool
		allTeamData          []*teamData
		nameByCarID          map[int]string
		nameByClassID        map[int]string
		overallStatsBySplit  map[int]*statData
		carStatsBySplit      map[int][]*statData
		carClassStatsBySplit map[int][]*statData
	}
)

func (c *collectSplitsDataCommand) run(ctx context.Context) error {
	app, err := util.InitApp()
	if err != nil {
		log.Error("failed to initialize app", log.ErrorField(err))
		return err
	}
	defer app.Close()
	logger := log.GetFromContext(ctx)
	logger.Info("starting splits data collection",
		log.Int("subsession_id", c.subsessionID))
	data, err := app.API.Get(
		strings.TrimSpace(fmt.Sprintf(`
			/data/results/get?subsession_id=%d
			`, c.subsessionID)))
	if err != nil {
		log.Error("failed to get event result data", log.ErrorField(err))
		return err
	}
	var eventResult irdata.EventResult
	err = json.Unmarshal(data, &eventResult)
	if err != nil {
		log.Error("failed to parse event result data", log.ErrorField(err))
		return err
	}
	c.nameByCarID = make(map[int]string)
	c.nameByClassID = make(map[int]string)
	// ensure the remaining data is fetched
	l, r := c.parseSplitCond(len(eventResult.AssociatedSubsessionIDs))

	for split, otherSubsessionID := range eventResult.AssociatedSubsessionIDs[l:r] {
		data, err := app.API.Get(
			strings.TrimSpace(fmt.Sprintf(`
			/data/results/get?subsession_id=%d
			`, otherSubsessionID)))
		if err != nil {
			log.Error("failed to get event result data", log.ErrorField(err))
			return err
		}
		err = json.Unmarshal(data, &eventResult)
		if err != nil {
			log.Error("failed to parse event result data", log.ErrorField(err))
			return err
		}
		c.allTeamData = append(c.allTeamData, c.processEventResult(split, &eventResult)...)
	}

	c.processSplits(c.allTeamData)
	c.outputResults()
	if c.plot {
		c.generatePlots()
	}
	return nil
}

func (c *collectSplitsDataCommand) parseSplitCond(totalSplits int) (left, right int) {
	left, right = 0, totalSplits
	if c.splitCond == "" {
		return left, right
	}

	parts := strings.Split(c.splitCond, ":")
	if len(parts) == 2 {
		if parts[0] != "" {
			left, _ = strconv.Atoi(parts[0])
		}
		if parts[1] != "" {
			right, _ = strconv.Atoi(parts[1])
		}
		return max(0, left), min(totalSplits, right)
	}
	return left, right
}

func (c *collectSplitsDataCommand) outputResults() {
	splits := lo.Keys(c.carClassStatsBySplit)
	sort.Ints(splits)
	for _, split := range splits {
		for _, component := range c.components {
			switch component {
			case "summary":
				c.outputSummary(split)
			case "class":
				c.outputClass(split)
			case "car":
				c.outputCar(split)
			}
		}
	}
}

func (c *collectSplitsDataCommand) outputSummary(split int) {
	s, ok := c.overallStatsBySplit[split]
	if !ok {
		log.Warn("no class stats found for split", log.Int("split", split))
		return
	}
	allBySplit := lo.GroupBy(c.allTeamData, func(d *teamData) int {
		return d.split
	})
	dRatings := lo.FlatMap(allBySplit[split], func(d *teamData, _ int) []int {
		return d.driverRatings
	})
	driverStats := c.collectMinMaxAvg(dRatings)
	fmt.Printf("split:%2d #teams:%2d #driver: %2d team: (%s) driver: (%s)\n",
		split, s.numTeams, s.numDrivers, s.ratings.ratingStats(),
		driverStats.ratingStats(),
	)
}

func (c *collectSplitsDataCommand) outputClass(split int) {
	s, ok := c.carClassStatsBySplit[split]
	if !ok {
		log.Warn("no class stats found for split", log.Int("split", split))
		return
	}
	maxNameLen := len(lo.MaxBy(s, func(a, b *statData) bool {
		return len(a.name) > len(b.name)
	}).name)

	walkIDs := lo.Map(s, func(stat *statData, _ int) int {
		return stat.id
	})
	slices.SortStableFunc(walkIDs, func(a, b int) int {
		return strings.Compare(c.nameByClassID[a], c.nameByClassID[b])
	})

	byClassID := lo.SliceToMap(s, func(stat *statData) (int, *statData) {
		return stat.id, stat
	})
	idOutput := func(id int) string {
		if c.showIDs {
			return fmt.Sprintf("classID: %d ", id)
		}
		return ""
	}
	format := fmt.Sprintf(
		"split:%%2d %%sclass:%%-%ds #teams:%%2d #driver: %%2d team: (%%s) driver: (%%s)\n",
		maxNameLen,
	)
	for _, classID := range walkIDs {
		d := byClassID[classID]
		fmt.Printf(format,
			split, idOutput(classID), d.name, d.numTeams, d.numDrivers,
			d.ratings.ratingStats(),
			d.driverRatings.ratingStats())
	}
}

func (c *collectSplitsDataCommand) outputCar(split int) {
	s, ok := c.carStatsBySplit[split]
	_ = s
	if !ok {
		log.Warn("no car stats found for split", log.Int("split", split))
		return
	}
	maxCarNameLen := len(lo.MaxBy(s, func(a, b *statData) bool {
		return len(a.name) > len(b.name)
	}).name)

	walkIDs := lo.Map(s, func(stat *statData, _ int) int {
		return stat.id
	})
	slices.SortStableFunc(walkIDs, func(a, b int) int {
		return strings.Compare(c.nameByCarID[a], c.nameByCarID[b])
	})

	byCarID := lo.SliceToMap(s, func(stat *statData) (int, *statData) {
		return stat.id, stat
	})
	idOutput := func(id int) string {
		if c.showIDs {
			return fmt.Sprintf("carID: %d ", id)
		}
		return ""
	}
	format := fmt.Sprintf(
		"split:%%2d %%scar:%%-%ds #teams:%%2d #driver: %%2d team: (%%s) driver: (%%s)\n",
		maxCarNameLen,
	)
	for _, carID := range walkIDs {
		d := byCarID[carID]
		fmt.Printf(format,
			split, idOutput(carID), d.name, d.numTeams, d.numDrivers,
			d.ratings.ratingStats(),
			d.driverRatings.ratingStats())
	}
}

func (c *collectSplitsDataCommand) generatePlots() {
	for _, component := range c.components {
		switch component {
		case "class":
			c.plotClass()
		case "car":
			c.plotCar()
		}
	}
}

func (c *collectSplitsDataCommand) plotClass() {
	p := plot.New()
	p.Title.Text = "Car Class iRating"
	p.X.Label.Text = "Split"
	p.Y.Label.Text = "iRating"

	splits := lo.Keys(c.carClassStatsBySplit)
	sort.Ints(splits)

	for _, classID := range lo.Keys(c.nameByClassID) {
		pts := make(plotter.XYs, len(splits))
		for i, split := range splits {
			stats, ok := c.carClassStatsBySplit[split]
			if !ok {
				continue
			}
			stat, ok := lo.Find(stats, func(s *statData) bool {
				return s.id == classID
			})
			if !ok {
				continue
			}
			if stat.ratings.avg <= 0 {
				continue
			}
			pts[i].X = float64(split)
			pts[i].Y = float64(stat.ratings.avg)
		}
		line, err := plotter.NewLine(pts)
		if err != nil {
			log.Error("failed to create line for plot", log.ErrorField(err))
			continue
		}
		line.Width = vg.Points(2)
		line.Dashes = []vg.Length{vg.Points(5), vg.Points(5)}

		line.Color = plotutil.Color(classID)
		p.Add(line)
		p.Legend.Add(c.nameByClassID[classID], line)
		p.Legend.Top = true
	}

	if err := p.Save(10*vg.Inch, 4*vg.Inch, "tmp/line_car_class_participation.png"); err != nil {
		log.Error("failed to save plot", log.ErrorField(err))
	}
}

func (c *collectSplitsDataCommand) plotCar() {
	p := plot.New()
	p.Title.Text = "Car iRating"
	p.X.Label.Text = "Split"
	p.Y.Label.Text = "iRating"

	splits := lo.Keys(c.carStatsBySplit)
	sort.Ints(splits)

	filteredCarIDs := lo.Filter(lo.Keys(c.nameByCarID), func(carID, _ int) bool {
		if len(c.carIDs) > 0 && !lo.Contains(c.carIDs, carID) {
			return false
		}
		return true
	})
	for _, carID := range filteredCarIDs {
		pts := make(plotter.XYs, len(splits))
		for i, split := range splits {
			stats, ok := c.carStatsBySplit[split]
			if !ok {
				continue
			}
			stat, ok := lo.Find(stats, func(s *statData) bool {
				return s.id == carID
			})
			if !ok {
				continue
			}
			if stat.ratings.avg <= 0 {
				continue
			}
			pts[i].X = float64(split)
			pts[i].Y = float64(stat.ratings.avg)
		}
		scatter, err := plotter.NewScatter(pts)
		if err != nil {
			log.Error("failed to create scatter for plot", log.ErrorField(err))
			continue
		}
		scatter.Radius = vg.Points(2)
		scatter.Color = plotutil.Color(carID)

		scatter.Color = plotutil.Color(carID)
		p.Add(scatter)
		p.Legend.Add(c.nameByCarID[carID], scatter)
		p.Legend.Top = true
	}

	if err := p.Save(10*vg.Inch, 4*vg.Inch, "tmp/line_car_participation.png"); err != nil {
		log.Error("failed to save plot", log.ErrorField(err))
	}
}

func (c *collectSplitsDataCommand) processSplits(all []*teamData) {
	allBySplit := lo.GroupBy(all, func(d *teamData) int {
		return d.split
	})
	splits := lo.Keys(allBySplit)
	sort.Ints(splits)
	c.overallStatsBySplit = make(map[int]*statData)
	c.carStatsBySplit = make(map[int][]*statData)
	c.carClassStatsBySplit = make(map[int][]*statData)
	for _, split := range splits {
		td := allBySplit[split]
		c.overallStatsBySplit[split] = c.collectStats(td)
		c.overallStatsBySplit[split].numDrivers = lo.SumBy(td, func(d *teamData) int {
			return d.num
		})

		byCarClassID := lo.GroupBy(
			lo.Filter(td, func(d *teamData, _ int) bool {
				if len(c.carClassIDs) > 0 && !lo.Contains(c.carClassIDs, d.carClassID) {
					return false
				}
				return true
			}), func(d *teamData) int {
				return d.carClassID
			})

		for carClassID, td := range byCarClassID {
			byCar := lo.GroupBy(lo.Filter(td, func(d *teamData, _ int) bool {
				if len(c.carIDs) > 0 && !lo.Contains(c.carIDs, d.carID) {
					return false
				}
				return true
			}), func(d *teamData) int {
				return d.carID
			})
			s := c.collectStats(td)
			s.id = carClassID
			s.name = c.nameByClassID[carClassID]
			c.carClassStatsBySplit[split] = append(c.carClassStatsBySplit[split], s)

			for carID, tdCar := range byCar {
				s := c.collectStats(tdCar)
				s.id = carID
				s.name = c.nameByCarID[carID]
				c.carStatsBySplit[split] = append(c.carStatsBySplit[split], s)
			}
		}
	}
}

func (c *collectSplitsDataCommand) collectStats(tdCar []*teamData) *statData {
	ratings := lo.Map(tdCar, func(d *teamData, _ int) int { return d.rating })
	driverRatings := lo.FlatMap(tdCar, func(d *teamData, _ int) []int {
		return d.driverRatings
	})
	ret := &statData{
		ratings:       c.collectMinMaxAvg(ratings),
		driverRatings: c.collectMinMaxAvg(driverRatings),
		numTeams:      len(tdCar),
		numDrivers: lo.SumBy(tdCar, func(item *teamData) int {
			return item.num
		}),
	}
	return ret
}

func (c *collectSplitsDataCommand) collectMinMaxAvg(raw []int) *minMaxAvg {
	ret := &minMaxAvg{
		avg: lo.MeanBy(raw, func(d int) int {
			return d
		}),
		min: lo.MinBy(raw, func(a, b int) bool {
			return a < b
		}),
		max: lo.MaxBy(raw, func(a, b int) bool {
			return a > b
		}),
		num: len(raw),
	}
	return ret
}

//nolint:whitespace // editor/linter issue
func (c *collectSplitsDataCommand) processEventResult(
	split int,
	er *irdata.EventResult,
) []*teamData {
	log.Debug("processing event result",

		log.Int("subsession_id", er.SubsessionID),
	)
	teamDataList := make([]*teamData, 0)
	for i := range er.SessionResults {
		sr := er.SessionResults[i]
		if sr.SimSessionType != 6 {
			continue
		}
		for j := range sr.Results {
			r := sr.Results[j]
			teamDataList = append(teamDataList, &teamData{
				carID:         r.CarID,
				carClassID:    r.CarClassID,
				split:         split,
				rating:        c.collectIratings(r.DriverResults),
				num:           len(r.DriverResults),
				driverRatings: c.collectDriverIratings(r.DriverResults),
				carNumber:     r.Livery.CarNumber,
			})
			c.nameByCarID[r.CarID] = r.CarName
			c.nameByClassID[r.CarClassID] = r.CarClassShortName
		}
	}
	return teamDataList
}

//nolint:whitespace // editor/linter issue
func (c *collectSplitsDataCommand) collectIratings(
	driverResults []irdata.EventSessionResultEntryDriver,
) int {
	validDriverResults := lo.Filter(driverResults,
		func(r irdata.EventSessionResultEntryDriver, _ int) bool {
			return r.OldIRating > 0
		})
	return lo.MeanBy(validDriverResults,
		func(r irdata.EventSessionResultEntryDriver) int {
			return r.OldIRating
		})
}

//nolint:whitespace // editor/linter issue
func (c *collectSplitsDataCommand) collectDriverIratings(
	driverResults []irdata.EventSessionResultEntryDriver,
) []int {
	validDriverResults := lo.Filter(driverResults,
		func(r irdata.EventSessionResultEntryDriver, _ int) bool {
			return r.OldIRating > 0
		})
	return lo.Map(validDriverResults,
		func(r irdata.EventSessionResultEntryDriver, _ int) int {
			return r.OldIRating
		})
}

func (s *minMaxAvg) ratingStats() string {
	return fmt.Sprintf("minIR: %5d avgIR: %5d maxIR: %5d",
		s.min, s.avg, s.max)
}
