package variants

import (
	"fmt"

	"github.com/zond/godip"
	"github.com/zond/godip/graph"
	"github.com/zond/godip/orders"
	"github.com/zond/godip/phase"
	"github.com/zond/godip/state"
	"github.com/zond/godip/variants/ancientmediterranean"
	"github.com/zond/godip/variants/beta/gatewaywest"
	"github.com/zond/godip/variants/beta/threekingdoms"
	"github.com/zond/godip/variants/canton"
	"github.com/zond/godip/variants/chaos"
	"github.com/zond/godip/variants/classical"
	"github.com/zond/godip/variants/classicalcrowded"
	"github.com/zond/godip/variants/coldwar"
	"github.com/zond/godip/variants/common"
	"github.com/zond/godip/variants/empiresandcoalitions"
	"github.com/zond/godip/variants/europe1939"
	"github.com/zond/godip/variants/fleetrome"
	"github.com/zond/godip/variants/franceaustria"
	"github.com/zond/godip/variants/hundred"
	"github.com/zond/godip/variants/italygermany"
	"github.com/zond/godip/variants/northseawars"
	"github.com/zond/godip/variants/pure"
	"github.com/zond/godip/variants/sengoku"
	"github.com/zond/godip/variants/twentytwenty"
	"github.com/zond/godip/variants/unconstitutional"
	"github.com/zond/godip/variants/vietnamwar"
	"github.com/zond/godip/variants/westernworld901"
	"github.com/zond/godip/variants/year1908"
	"github.com/zond/godip/variants/youngstownredux"
)

func init() {
	for _, variant := range OrderedVariants {
		Variants[variant.Name] = variant
	}
}

type VariantData struct {
	Nations               []godip.Nation
	Regions               map[godip.Province]RegionData
	StartingSupplyCenters map[godip.Province]godip.Nation
	StartingUnits         map[godip.Province]godip.Unit
	StartingYear          int
}

type RegionData struct {
	IsSupplyCenter bool
	RegionType     []godip.Flag
	Neighbors      map[godip.Province][]godip.Flag
}

var (
	PhaseTypes = []godip.PhaseType{godip.Movement, godip.Retreat, godip.Adjustment}
	Seasons    = []godip.Season{godip.Spring, godip.Fall}
	UnitTypes  = []godip.UnitType{godip.Army, godip.Fleet}
	Parser     = orders.NewParser([]godip.Order{
		orders.BuildOrder,
		orders.ConvoyOrder,
		orders.DisbandOrder,
		orders.HoldOrder,
		orders.MoveOrder,
		orders.MoveViaConvoyOrder,
		orders.SupportOrder,
	})
)

var FromJson = func(variantData VariantData) common.Variant {
	return common.Variant{
		Name:  "From JSON",
		Start: GetStart(variantData),
		Blank: GetBlank(variantData),
		BlankStart: func() (result *state.State, err error) {
			result = GetBlank(variantData)(NewPhase(variantData.StartingYear-1, godip.Fall, godip.Adjustment))
			return
		},
		Parser:      Parser,
		Graph:       func() godip.Graph { return BuildGraph(variantData) },
		Phase:       NewPhase,
		Nations:     variantData.Nations,
		PhaseTypes:  PhaseTypes,
		Seasons:     Seasons,
		UnitTypes:   UnitTypes,
		SoloWinner:  common.SCCountWinner(GetMajoritySupplyCenterCount(variantData)),
		SoloSCCount: func(*state.State) int { return GetMajoritySupplyCenterCount(variantData) },
	}
}

func BuildGraph(variantData VariantData) *graph.Graph {
	graph := graph.New()
	for province, regionData := range variantData.Regions {
		subnode := graph.Prov(province)
		for nProv, flags := range regionData.Neighbors {
			subnode = subnode.Conn(nProv, flags...)
		}
		subnode = subnode.Flag(regionData.RegionType...)
		startingSc, ok := variantData.StartingSupplyCenters[province]
		if ok {
			subnode.SC(startingSc)
		}
	}
	return graph
}

func AdjustSCs(phase *phase.Phase) bool {
	return phase.Ty == godip.Retreat && phase.Se == godip.Fall
}

func NewPhase(year int, season godip.Season, typ godip.PhaseType) godip.Phase {
	return phase.Generator(Parser, AdjustSCs)(year, season, typ)
}

func GetStart(variantData VariantData) func() (result *state.State, err error) {
	return func() (result *state.State, err error) {
		result = GetBlank(variantData)(NewPhase(variantData.StartingYear, godip.Spring, godip.Movement))
		if err = result.SetUnits(variantData.StartingUnits); err != nil {
			return
		}
		result.SetSupplyCenters(variantData.StartingSupplyCenters)
		return
	}
}

func GetBlank(variantData VariantData) func(phase godip.Phase) *state.State {
	return func(phase godip.Phase) *state.State {
		return state.New(BuildGraph(variantData), phase, BackupRule, nil, nil)
	}
}

func GetMajoritySupplyCenterCount(variantData VariantData) int {
	numScs := 0
	for _, regionData := range variantData.Regions {
		if regionData.IsSupplyCenter {
			numScs++
		}
	}

	majorityScs := 0
	for majorityScs <= numScs/2 {
		majorityScs++
	}
	return majorityScs
}

func BackupRule(state godip.State, deps []godip.Province) (err error) {
	only_moves := true
	convoys := false
	for _, prov := range deps {
		if order, _, ok := state.Order(prov); ok {
			if order.Type() != godip.Move {
				only_moves = false
			}
			if order.Type() == godip.Convoy {
				convoys = true
			}
		}
	}

	if only_moves {
		for _, prov := range deps {
			state.SetResolution(prov, nil)
		}
		return
	}
	if convoys {
		for _, prov := range deps {
			if order, _, ok := state.Order(prov); ok && order.Type() == godip.Convoy {
				state.SetResolution(prov, godip.ErrConvoyParadox)
			}
		}
		return
	}

	err = fmt.Errorf("unknown circular dependency between %v", deps)
	return
}

var Variants = map[string]common.Variant{}

var OrderedVariants = []common.Variant{
	gatewaywest.GatewayWestVariant,
	classicalcrowded.ClassicalCrowdedVariant,
	threekingdoms.ThreeKingdomsVariant,
	ancientmediterranean.AncientMediterraneanVariant,
	canton.CantonVariant,
	chaos.ChaosVariant,
	classical.ClassicalVariant,
	coldwar.ColdWarVariant,
	empiresandcoalitions.EmpiresAndCoalitionsVariant,
	europe1939.Europe1939Variant,
	fleetrome.FleetRomeVariant,
	franceaustria.FranceAustriaVariant,
	hundred.HundredVariant,
	italygermany.ItalyGermanyVariant,
	northseawars.NorthSeaWarsVariant,
	pure.PureVariant,
	sengoku.SengokuVariant,
	twentytwenty.TwentyTwentyVariant,
	unconstitutional.UnconstitutionalVariant,
	vietnamwar.VietnamWarVariant,
	westernworld901.WesternWorld901Variant,
	year1908.Year1908Variant,
	youngstownredux.YoungstownReduxVariant,
}
