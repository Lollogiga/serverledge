package energyStore

import "fmt"

// Costruiamo la chiave etcd per una variante.
// functionName = nome della variante serveless (fn.Name)
// variantID = id della variante
func EnergyKey(functionName, variantID string) string {
	return fmt.Sprintf("/energy/%s/%s", functionName, variantID)

}
