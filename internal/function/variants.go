package function

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// generateVariants prende la funzione base e produce light, medium e heavy.
func GenerateVariants(base Function) ([]Function, error) {

	envLight := map[string]string{}
	envMedium := map[string]string{}
	envHeavy := map[string]string{}

	for key, raw := range base.ApproxConfig {

		switch v := raw.(type) {

		// Caso: min/max → generazione automatica numerica
		case map[string]interface{}:

			hasMin := v["min"]
			hasMax := v["max"]
			hasLight := v["light"]
			hasMedium := v["medium"]
			hasHeavy := v["heavy"]

			// Caso 1: numerico min/max
			if hasMin != nil && hasMax != nil {
				min := toFloat(v["min"])
				max := toFloat(v["max"])
				mid := math.Sqrt(min * max)

				envLight["APPROX_"+strings.ToUpper(key)] = fmt.Sprintf("%g", min)
				envMedium["APPROX_"+strings.ToUpper(key)] = fmt.Sprintf("%g", mid)
				envHeavy["APPROX_"+strings.ToUpper(key)] = fmt.Sprintf("%g", max)
				continue
			}

			// Caso 2: valori espliciti light/medium/heavy
			if hasLight != nil && hasMedium != nil && hasHeavy != nil {
				envLight["APPROX_"+strings.ToUpper(key)] = stringify(v["light"])
				envMedium["APPROX_"+strings.ToUpper(key)] = stringify(v["medium"])
				envHeavy["APPROX_"+strings.ToUpper(key)] = stringify(v["heavy"])
				continue
			}

		// Caso: valore singolo → copia su tutte le varianti
		default:
			common := stringify(v)
			envLight["APPROX_"+strings.ToUpper(key)] = common
			envMedium["APPROX_"+strings.ToUpper(key)] = common
			envHeavy["APPROX_"+strings.ToUpper(key)] = common
		}
	}

	// Costruisci i 3 cloni
	light := base
	medium := base
	heavy := base

	light.Name = base.Name + "_light"
	medium.Name = base.Name + "_medium"
	heavy.Name = base.Name // default = heavy

	light.Env = envLight
	medium.Env = envMedium
	heavy.Env = envHeavy

	return []Function{light, medium, heavy}, nil
}

func stringify(v interface{}) string {
	return fmt.Sprintf("%v", v)
}

func toFloat(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	default:
		return 0.0
	}
}
