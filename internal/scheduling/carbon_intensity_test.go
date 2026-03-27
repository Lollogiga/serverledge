package scheduling

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func writeTempCSV(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "zone_ci_params_*.csv")
	if err != nil {
		t.Fatalf("cannot create temp CSV: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("cannot write temp CSV: %v", err)
	}
	f.Close()
	return f.Name()
}

func approxEqual(a, b, tol float64) bool {
	return math.Abs(a-b) < tol
}

// --- loadZoneParamsCSV ---

func TestLoadZoneParamsCSV_ValidRows(t *testing.T) {
	content := "# commento\nzone,ci_mid,s,k\nDE,277.94,41.3874,0.45\nIT,290.00,45.0000,0.50\n"
	path := writeTempCSV(t, content)
	m, err := loadZoneParamsCSV(path)
	if err != nil {
		t.Fatalf("errore: %v", err)
	}
	cases := []struct {
		zone  string
		ciMid float64
		s     float64
		k     float64
	}{
		{"DE", 277.94, 41.3874, 0.45},
		{"IT", 290.00, 45.0000, 0.50},
	}
	for _, c := range cases {
		p, ok := m[c.zone]
		if !ok {
			t.Errorf("zona %q non trovata", c.zone)
			continue
		}
		if !approxEqual(p.CIMid, c.ciMid, 0.01) {
			t.Errorf("%s: CIMid atteso %.2f, ottenuto %.2f", c.zone, c.ciMid, p.CIMid)
		}
		if !approxEqual(p.S, c.s, 0.0001) {
			t.Errorf("%s: S atteso %.4f, ottenuto %.4f", c.zone, c.s, p.S)
		}
		if !approxEqual(p.K, c.k, 0.001) {
			t.Errorf("%s: K atteso %.3f, ottenuto %.3f", c.zone, c.k, p.K)
		}
	}
}

func TestLoadZoneParamsCSV_MissingKColumn(t *testing.T) {
	content := "zone,ci_mid,s\nDE,277.94,41.3874\n"
	path := writeTempCSV(t, content)
	m, err := loadZoneParamsCSV(path)
	if err != nil {
		t.Fatalf("errore: %v", err)
	}
	p, ok := m["DE"]
	if !ok {
		t.Fatal("zona DE non trovata")
	}
	if !approxEqual(p.K, ciK, 0.001) {
		t.Errorf("K atteso default %.3f, ottenuto %.3f", ciK, p.K)
	}
}

func TestLoadZoneParamsCSV_FileNotFound(t *testing.T) {
	_, err := loadZoneParamsCSV("/tmp/non_esiste_xyz.csv")
	if err == nil {
		t.Error("atteso errore per file mancante, ottenuto nil")
	}
}

// --- CarbonIntensityToLambdaWithStats ---

func TestLambda_AtMidpoint(t *testing.T) {
	lambda := CarbonIntensityToLambdaWithStats(277.94, 277.94, 41.3874, 0.45)
	if !approxEqual(lambda, 0.5, 1e-9) {
		t.Errorf("CI=ci_mid: atteso λ=0.500, ottenuto λ=%.6f", lambda)
	}
}

func TestLambda_CleanGrid(t *testing.T) {
	// CI molto bassa rispetto alla media: la sigmoide con k=0.45 da' ~0.92.
	// Verifichiamo che lambda sia chiaramente nella zona "pulita" (>0.85).
	lambda := CarbonIntensityToLambdaWithStats(50.0, 277.94, 41.3874, 0.45)
	if lambda < 0.85 {
		t.Errorf("CI bassa: atteso lambda>=0.85, ottenuto lambda=%.4f", lambda)
	}
}

func TestLambda_DirtyGrid(t *testing.T) {
	// CI molto alta rispetto alla media: la sigmoide con k=0.45 da' ~0.03.
	// Verifichiamo che lambda sia chiaramente nella zona "sporca" (<0.15).
	lambda := CarbonIntensityToLambdaWithStats(600.0, 277.94, 41.3874, 0.45)
	if lambda > 0.15 {
		t.Errorf("CI alta: atteso lambda<=0.15, ottenuto lambda=%.4f", lambda)
	}
}

func TestLambda_RangeCheck(t *testing.T) {
	for _, ci := range []float64{0, 50, 100, 200, 277.94, 350, 500, 800} {
		lambda := CarbonIntensityToLambdaWithStats(ci, 277.94, 41.3874, 0.45)
		if lambda <= 0 || lambda >= 1 {
			t.Errorf("CI=%.0f: lambda=%.6f fuori da (0,1)", ci, lambda)
		}
	}
}

func TestLambda_KEffect(t *testing.T) {
	ci, ciMid, s := 350.0, 277.94, 41.3874
	lambdaLowK := CarbonIntensityToLambdaWithStats(ci, ciMid, s, 0.4)
	lambdaHighK := CarbonIntensityToLambdaWithStats(ci, ciMid, s, 1.0)
	if lambdaLowK <= lambdaHighK {
		t.Errorf("k alto dovrebbe dare lambda piu basso: k=0.4->%.4f, k=1.0->%.4f", lambdaLowK, lambdaHighK)
	}
}

// --- integrazione con il CSV reale ---

func TestLoadRealZoneParamsCSV(t *testing.T) {
	candidates := []string{
		"../../zone_ci_params.csv",
		filepath.Join(os.Getenv("PWD"), "zone_ci_params.csv"),
	}
	var path string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			path = c
			break
		}
	}
	if path == "" {
		t.Skip("zone_ci_params.csv non trovato")
	}

	m, err := loadZoneParamsCSV(path)
	if err != nil {
		t.Fatalf("errore lettura CSV reale: %v", err)
	}
	if len(m) == 0 {
		t.Error("nessuna zona caricata")
	}

	t.Logf("Zone caricate (%d):", len(m))
	for zone, p := range m {
		lambda := CarbonIntensityToLambdaWithStats(p.CIMid, p.CIMid, p.S, p.K)
		if !approxEqual(lambda, 0.5, 1e-9) {
			t.Errorf("zona %s: lambda al punto medio = %.6f (atteso 0.5)", zone, lambda)
		}
		t.Logf("  %-5s ci_mid=%.1f  s=%.4f  k=%.2f  lambda(ci_mid)=%.4f", zone, p.CIMid, p.S, p.K, lambda)
	}
}
