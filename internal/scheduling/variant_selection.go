package scheduling

import (
	"errors"
	"log"
	"math"
	"strings"

	"github.com/serverledge-faas/serverledge/internal/config"
	"github.com/serverledge-faas/serverledge/internal/function"
	"github.com/serverledge-faas/serverledge/internal/node"
)

const energyEpsilonRatio = 0.05 // 5%
const worstErrorScore = 1e12

func energyEpsilon(budget float64) float64 {
	return budget * energyEpsilonRatio
}

func energyCostJoule(fn *function.Function, warm bool) (float64, error) {
	if fn.EnergyProfile == nil {
		return worstErrorScore, errors.New("missing energy profile")
	}

	includeColdStart := config.GetBool(
		config.SchedulingEnergyIncludeColdStart,
		true,
	)

	if warm {
		return fn.EnergyProfile.InvocationJoule, nil
	}

	if includeColdStart {
		return fn.EnergyProfile.ColdStartJoule +
			fn.EnergyProfile.InvocationJoule, nil
	}

	return fn.EnergyProfile.InvocationJoule, nil
}

func errorScore(fn *function.Function) float64 {
	if fn.OutputModel == nil {
		// Legacy o non specificato: trattiamo come esatto
		return 0.0
	}

	switch fn.OutputModel.Type {

	case "error":
		if fn.OutputModel.ErrorEstimate != nil {
			return *fn.OutputModel.ErrorEstimate
		}
		return worstErrorScore

	case "quality":
		if fn.OutputModel.Quality != nil {
			switch strings.ToLower(*fn.OutputModel.Quality) {
			case "high":
				return 0.0
			case "medium":
				return 1.0
			case "low":
				return 2.0
			}
		}
		return worstErrorScore
	}

	// Tipo sconosciuto → pessimistico
	return worstErrorScore
}

func SelectEnergyAwareVariant(
	r *function.Request,
) (*function.Function, *function.VariantSchedulingReport, error) {

	if r == nil || r.Fun == nil {
		return nil, nil, errors.New("invalid request")
	}

	base := r.Fun

	report := &function.VariantSchedulingReport{
		LogicalName:     base.LogicalName,
		InvokedFunction: base.Name,
		AllowApprox:     r.AllowApprox,
	}

	if r.MaxEnergyJoule != nil {
		report.MaxEnergyJoule = *r.MaxEnergyJoule
	}

	// Recupera tutte le varianti tramite indice
	variants, err := function.GetFunctionsByLogicalName(base.LogicalName)
	if err != nil || len(variants) == 0 {
		report.DecisionReason = "fallback-base"
		return base, report, nil
	}

	// ===============================
	// PHASE 1: evaluate all variants
	// ===============================
	type evaluatedVariant struct {
		fn       *function.Function
		warm     bool
		energy   float64
		errScore float64
	}

	var evaluated []evaluatedVariant

	for _, fn := range variants {
		if fn == nil {
			continue
		}

		// Warm hint (predittivo, non vincolante)
		warm := node.HasWarmContainer(fn)

		// Energia stimata
		energy, err := energyCostJoule(fn, warm)
		if err != nil {
			continue
		}

		// Errore stimato
		errVal := errorScore(fn)

		evaluated = append(evaluated, evaluatedVariant{
			fn:       fn,
			warm:     warm,
			energy:   energy,
			errScore: errVal,
		})
	}

	if len(evaluated) == 0 {
		report.DecisionReason = "fallback-base"
		return base, report, nil
	}

	// ===============================
	// PHASE 2: selection logic
	// ===============================

	// -------- CASE A: no energy budget --------
	if r.MaxEnergyJoule == nil || *r.MaxEnergyJoule <= 0 {

		best := evaluated[0]

		for i := 1; i < len(evaluated); i++ {
			cur := evaluated[i]

			// 1) errore minore
			if cur.errScore < best.errScore {
				best = cur
				continue
			}
			if cur.errScore > best.errScore {
				continue
			}

			// 2) energia minore
			if cur.energy < best.energy {
				best = cur
			}
		}

		report.SelectedFunction = best.fn.Name
		report.VariantID = best.fn.VariantID
		report.EstimatedEnergy = best.energy
		report.WarmHint = best.warm
		report.ErrorEstimate = best.errScore
		report.DecisionReason = "error-first"

		return best.fn, report, nil
	}

	// -------- CASE B: energy budget present --------
	budget := *r.MaxEnergyJoule
	log.Print("Energy budget:", budget)
	eps := energyEpsilon(budget)

	var feasible []evaluatedVariant
	for _, v := range evaluated {
		if v.energy <= budget+eps {
			feasible = append(feasible, v)
		}
	}

	// Nessuna variante compatibile
	if len(feasible) == 0 {
		report.DecisionReason = "no-feasible-variants"
		return base, report, errors.New("no variants satisfying energy budget")
	}

	best := feasible[0]

	for i := 1; i < len(feasible); i++ {
		cur := feasible[i]

		// 1) errore minore
		if cur.errScore < best.errScore {
			best = cur
			continue
		}
		if cur.errScore > best.errScore {
			continue
		}

		// 2) best-fit energetico (più vicino al budget)
		if math.Abs(budget-cur.energy) < math.Abs(budget-best.energy) {
			best = cur
		}
	}

	report.SelectedFunction = best.fn.Name
	report.VariantID = best.fn.VariantID
	report.EstimatedEnergy = best.energy
	report.WarmHint = best.warm
	report.ErrorEstimate = best.errScore
	report.DecisionReason = "energy-budget"

	return best.fn, report, nil
}
