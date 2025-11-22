package function

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

func GenerateVariants(base Function) ([]Function, error) {
	if base.ApproxConfig == nil || len(base.ApproxConfig) == 0 {
		return nil, errors.New("no approx_config provided")
	}

	light := cloneFunction(base)
	medium := cloneFunction(base)
	heavy := cloneFunction(base)

	light.Name = base.Name + "_light"
	medium.Name = base.Name + "_medium"
	heavy.Name = base.Name

	light.Env["APPROX_VARIANT"] = "light"
	medium.Env["APPROX_VARIANT"] = "medium"
	heavy.Env["APPROX_VARIANT"] = "heavy"

	for key, raw := range base.ApproxConfig {

		cfg, ok := raw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid approx_config format for %s", key)
		}

		min, minOk := cfg["min"]
		max, maxOk := cfg["max"]

		if minOk && maxOk {
			minF := toFloat(min)
			maxF := toFloat(max)
			mid := math.Sqrt(minF * maxF)

			envKey := "APPROX_" + toEnvName(key)

			light.Env[envKey] = fmt.Sprintf("%g", minF)
			medium.Env[envKey] = fmt.Sprintf("%g", mid)
			heavy.Env[envKey] = fmt.Sprintf("%g", maxF)

			continue
		}

		if l, ok := cfg["light"]; ok {
			light.Env["APPROX_"+toEnvName(key)] = fmt.Sprintf("%v", l)
		}
		if m, ok := cfg["medium"]; ok {
			medium.Env["APPROX_"+toEnvName(key)] = fmt.Sprintf("%v", m)
		}
		if h, ok := cfg["heavy"]; ok {
			heavy.Env["APPROX_"+toEnvName(key)] = fmt.Sprintf("%v", h)
		}
	}

	return []Function{light, medium, heavy}, nil
}

// -------------------------
// HELPERS
// -------------------------

func cloneFunction(f Function) Function {
	nf := f

	// copia profonda delle variabili d’ambiente
	nf.Env = make(map[string]string)
	if f.Env != nil {
		for k, v := range f.Env {
			nf.Env[k] = v
		}
	}

	return nf
}

func toEnvName(s string) string {
	return strings.ToUpper(strings.ReplaceAll(s, " ", "_"))
}

func toFloat(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	default:
		panic("value is not numeric")
	}
}
