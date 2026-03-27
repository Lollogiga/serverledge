# Report sulla Selezione di Varianti con UCB e Ottimizzazione Energetica in Serverledge

**Tesi:** Schedulazione Green-Aware di Funzioni Serverless tramite Selezione di Varianti Pareto-Ottimale con Esplorazione UCB  
**Sistema:** Serverledge FaaS (Function-as-a-Service) — Edge Computing  
**Data:** 2025

---

## 1. Motivazione

Il paradigma FaaS su infrastrutture edge introduce una tensione fondamentale tra due obiettivi
che sono spesso in conflitto:

1. **Precisione del risultato** — l'utente si aspetta output corretti o di alta qualità.  
2. **Consumo energetico** — il nodo edge ha risorse limitate e opera in un contesto
   in cui il costo ambientale dell'energia può variare significativamente nel tempo a seconda
   di quanta energia rinnovabile è disponibile nella rete elettrica locale.

Per alcune classi di funzioni esistono *varianti approssimate* che producono risultati
leggermente meno precisi ma richiedono meno energia computazionale. Un sistema green-aware
dovrebbe essere capace di scegliere automaticamente la variante più adeguata al contesto
corrente: preferire l'accuratezza massima quando la rete è alimentata da fonti rinnovabili,
e privilegiare il risparmio energetico quando l'intensità di carbonio è elevata.

Il presente lavoro estende Serverledge con:

- Una **selezione di varianti Pareto-ottimale** guidata dal parametro λ derivato
  dall'intensità di carbonio della rete elettrica locale (ElectricityMaps API).
- Una **funzione sigmoide parametrizzata per zona geografica**, con parametri
  pre-calcolati offline da dati storici.
- Un meccanismo di **esplorazione UCB (Upper Confidence Bound)** che promuove
  la valutazione di varianti poco osservate, aggiornando le stime energetiche
  con dati reali raccolti da InfluxDB.

---

## 2. Architettura della Selezione di Varianti

### 2.1 Struttura delle varianti

Ogni funzione logica (es. `fibonacci`, `Sqrt`, `PiLeibniz`) può avere più *varianti fisiche*
registrate in Serverledge, descritte da un file `<nome>.json`:

```json
{
  "id": "binet",
  "runtime": "python310",
  "entry_point": "fibonacci_binet.handler",
  "src": "fibonacci_binet.py",
  "energy": {
    "cold_start_joule": 0.050,
    "invocation_joule": 0.000148
  },
  "output": {
    "type": "error",
    "error_estimate": 0.001
  },
  "is_approximate": true
}
```

I due obiettivi da ottimizzare sono:

| Campo | Descrizione |
|-------|-------------|
| `energy.invocation_joule` | Energia stimata per una singola invocazione (J) |
| `output.error_estimate`   | Errore relativo stimato dell'output (0 = esatto) |

Per funzioni ML (`SentimentAnalysis`, `ImageClassification`), l'output è caratterizzato da
una `quality` (`"high"`, `"medium"`, `"low"`) che viene mappata su un punteggio numerico
(0, 1, 2) per consentire la comparazione.

### 2.2 Frontiera di Pareto

Dati `n` varianti, il sistema calcola la frontiera di Pareto bi-obiettivo
(energia, errore), entrambi da minimizzare, usando l'algoritmo:

1. **Ordinamento** per energia crescente; pareggi rotti per errore crescente — O(n log n).
2. **Scansione singola**: un punto entra nella frontiera se e solo se il suo errore è
   *strettamente minore* del minimo errore dei punti precedenti — O(n).

L'algoritmo garantisce l'identificazione di tutti e soli i punti non dominati in O(n log n)
totale, ottimale per 2 obiettivi.

**Esempio (Fibonacci, container warm):**

| Variante | Energia (J) | Errore | Pareto? |
|----------|-------------|--------|---------|
| `binet` | 0.000148 | 0.001 | ✓ |
| `c`     | 0.000172 | 0.000 | ✓ |
| `optimization-py` | 0.000495 | 0.000 | ✗ (dominata da `c`) |
| `base`  | 0.000732 | 0.000 | ✗ (dominata da `c`) |

La frontiera di Pareto è `{binet, c}`. La scelta tra le due dipende da λ.

### 2.3 Scalarizzazione con λ

Data la frontiera di Pareto, ogni variante riceve un punteggio:

$$\text{score}(i) = (1 - \lambda) \cdot \hat{E}_i + \lambda \cdot \hat{\varepsilon}_i$$

dove $\hat{E}_i$ e $\hat{\varepsilon}_i$ sono i valori di energia ed errore normalizzati
sull'intervallo [0, 1] rispetto al range della frontiera.

La variante con il **punteggio minimo** viene selezionata.

- **λ → 0** (rete sporca, alta CI): peso sull'energia → variante più efficiente selezionata
  (es. `binet` per Fibonacci).
- **λ → 1** (rete pulita, bassa CI): peso sull'accuratezza → variante più precisa selezionata
  (es. `c` per Fibonacci).
- **λ = 0.5**: selezione al "ginocchio" della frontiera di Pareto.

---

## 3. Calcolo di λ dalla Carbon Intensity

### 3.1 Funzione sigmoide per zona geografica

Per ciascun nodo edge, la carbon intensity (CI) corrente viene ottenuta in tempo reale
dall'API ElectricityMaps (cache 15 min). Il parametro λ è derivato tramite una sigmoide
logistica centrata sui valori storici della zona:

$$\lambda = \frac{1}{1 + \exp\!\left(k \cdot \frac{CI - \mu_z}{\sigma_z}\right)}$$

dove:

| Simbolo | Significato |
|---------|-------------|
| $CI$ | Intensità di carbonio corrente (gCO₂eq/kWh) |
| $\mu_z$ | Media storica per la zona $z$ (CI media del periodo) |
| $\sigma_z$ | Parametro di scala, calcolato da $\sigma_z = (P_{95} - P_5)/(2 \ln\frac{1-\alpha}{\alpha})$ con $\alpha=0.1$ |
| $k$ | Parametro di pendenza per zona (di default 0.45) |

La formula per $\sigma_z$ deriva dalla definizione: si vuole che $\lambda(P_{95}) = \alpha$,
ossia che sulle code della distribuzione storica λ raggiunga i valori estremi di $\alpha$ e $1-\alpha$.

### 3.2 Parametri per zona

I parametri sono pre-calcolati offline dallo script `ScriptTesi/carbon_lambda_analysis.py`
utilizzando i dati storici di ElectricityMaps e salvati in `zone_ci_params.csv`:

| Zona | μ (gCO₂/kWh) | σ | k |
|------|--------------|---|---|
| DE   | 277.94 | 41.39 | 0.45 |
| ES   | 95.14  | 13.50 | 0.45 |
| FR   | 13.81  | 3.87  | 0.45 |
| NO   | 6.01   | 1.03  | 0.45 |
| PL   | 498.52 | 35.28 | 0.45 |
| IT   | 290.00 | 45.00 | 0.45 |
| GB   | 175.00 | 27.00 | 0.45 |

**Interpretazione:** in Francia (prevalenza nucleare), la CI media è ~14 gCO₂/kWh
e λ è quasi sempre vicino a 1 (priorità all'accuratezza). In Polonia (prevalenza carbone),
la CI media è ~499 gCO₂/kWh e λ è quasi sempre vicino a 0 (priorità all'energia).

---

## 4. Esplorazione UCB (Upper Confidence Bound)

### 4.1 Motivazione

Il profilo energetico iniziale di ogni variante è derivato da misurazioni offline
(profiling su hardware specifico). Tuttavia:

1. **L'energia reale dipende dall'hardware del nodo**: un nodo edge con CPU diversa
   avrà consumi differenti.
2. **Le stime iniziali possono essere imprecise**: il profiling offline potrebbe non
   riflettere il carico medio di produzione.
3. **L'EMA (Exponential Moving Average)** aggiorna la stima nel tempo, ma converge
   lentamente e non tiene conto dell'incertezza statistica.

Il meccanismo UCB affronta questo problema con principio di **ottimismo sotto incertezza**
(Optimism in the Face of Uncertainty, OFU): per varianti poco esplorate, il sistema
assume ottimisticamente che il consumo energetico reale possa essere inferiore alla
stima corrente. Questo incentiva l'esplorazione di varianti sotto-campionate.

### 4.2 Formula LCB (Lower Confidence Bound)

Per la minimizzazione energetica, si usa il limite inferiore di confidenza:

$$\text{LCB}_i = \mu_i - \beta \cdot \frac{\sigma_i}{\sqrt{n_i}}$$

dove:

| Simbolo | Significato |
|---------|-------------|
| $\mu_i$ | Media energetica della variante $i$ (da InfluxDB, ultimi 24h) |
| $\sigma_i$ | Deviazione standard energetica |
| $n_i$ | Numero di campioni InfluxDB |
| $\beta$ | Parametro di esplorazione (configurabile, default 1.0) |

Il LCB è equivalente all'UCB per massimizzazione applicato alla quantità $-E$ (energia
negata), ed è la funzione di acquisizione standard per la minimizzazione in ottimizzazione
bayesiana.

**Proprietà:**
- Se $n_i$ è grande (variante ben esplorata): $\text{LCB}_i \to \mu_i$ (sfruttamento)
- Se $n_i$ è piccolo (variante sotto-esplorata): $\text{LCB}_i \ll \mu_i$ (esplorazione)
- $\beta = 0$: degenerazione in pure exploitation (UCB disabilitato)
- $\beta \to \infty$: pura esplorazione (ignorando la media)

### 4.3 Integrazione nel selettore Pareto

```
Per ogni variante i:
  Se n_i ≥ min_samples AND β > 0:
    energy_i ← LCB_i  (ottimistico, promuove esplorazione)
  Altrimenti:
    energy_i ← etcd_value_i  (fallback deterministico)

Calcolare frontiera di Pareto su {energy_i, error_i}
Scalarizzare con λ → selezionare variante ottimale
```

Il fallback al valore etcd garantisce la retrocompatibilità: se InfluxDB non è
configurato o la variante ha meno di `min_samples` osservazioni (default: 5),
il sistema si comporta esattamente come prima.

### 4.4 Configurazione

In `serverledge-conf.yaml`:

```yaml
scheduling:
  ucb:
    beta: 1.0       # 0 = disabilita UCB; >1 = più esplorazione
    min_samples: 5  # campioni minimi prima di usare LCB
```

### 4.5 Trasparenza nel report di scheduling

Ogni invocazione include nel campo `variant_scheduling` della risposta JSON i dettagli
della selezione, incluse le informazioni UCB:

```json
{
  "variant_scheduling": {
    "logical_name": "fibonacci",
    "selected_function": "fibonacci-binet",
    "variant_id": "binet",
    "quality_weight": 0.23,
    "carbon_intensity_gco2": 310.5,
    "ucb_active": true,
    "pareto_front": [
      {
        "variant_id": "binet",
        "energy_joule": 0.000132,
        "error_estimate": 0.001,
        "ucb_exploration": true,
        "ucb_sample_count": 12
      },
      {
        "variant_id": "c",
        "energy_joule": 0.000171,
        "error_estimate": 0.0,
        "ucb_exploration": false,
        "ucb_sample_count": 0
      }
    ]
  }
}
```

---

## 5. Descrizione delle Varianti

### 5.1 Fibonacci

Il calcolo di Fibonacci genera la sequenza $F_0, F_1, \ldots, F_n$.

| Variante | Runtime | Algoritmo | Energia Inv. (J) | Errore | Approssimata |
|----------|---------|-----------|-----------------|--------|--------------|
| `base` | Python 3.10 | Iterativo con concatenazione stringa | 0.000732 | 0.0 | No |
| `optimization-py` | Python 3.10 | Iterativo con lista + join | 0.000495 | 0.0 | No |
| `c` | Native (C) | Iterativo in C | 0.000172 | 0.0 | No |
| `binet` | Python 3.10 | Formula di Binet $F_k = \text{round}(\varphi^k / \sqrt{5})$ | 0.000148 | 0.001 | **Sì** |

**Formula di Binet:**
$$F_k = \left\lfloor \frac{\varphi^k}{\sqrt{5}} + \frac{1}{2} \right\rfloor, \quad \varphi = \frac{1 + \sqrt{5}}{2}$$

Questa formula è O(1) per termine (vs O(k) iterativo). Per $k \leq 70$ è esatta in
double precision IEEE-754. Per $k > 70$, la propagazione dell'errore floating-point
di $\varphi^k$ introduce uno scarto di $\pm 1$ sul risultato intero. L'`error_estimate`
di 0.001 riflette questo regime per input tipici ($n \approx 50-200$).

**Frontiera di Pareto (warm):** `{binet, c}`  
**Selezione:** `binet` per λ < 0.5 (rete sporca), `c` per λ > 0.5 (rete pulita).

### 5.2 PiLeibniz

Serie di Leibniz per π: $\pi/4 = \sum_{k=0}^{N-1} \frac{(-1)^k}{2k+1}$.
L'accuratezza cresce con N.

| Variante | N termini | Energia Inv. (J) | Errore relativo |
|----------|-----------|-----------------|-----------------|
| `base` | adattivo | 0.000xxx | 0.0 |
| `n1000` | 1.000 | basso | 0.002 |
| `n5000` | 5.000 | medio-basso | 0.0004 |
| `n10000` | 10.000 | medio | 0.0002 |
| `n50000` | 50.000 | medio-alto | 4×10⁻⁵ |
| `n100000` | 100.000 | alto | 2×10⁻⁵ |
| `n200000` | 200.000 | molto alto | 1×10⁻⁵ |

Con 9 varianti, PiLeibniz offre la frontiera di Pareto più ricca, coprendo l'intero
spettro accuratezza/energia.

### 5.3 Sqrt

Calcolo della radice quadrata.

| Variante | Algoritmo | Energia Inv. (J) | Errore relativo |
|----------|-----------|-----------------|-----------------|
| `base` | `math.sqrt()` (Python) | 0.000453 | 0.0 |
| `Light` | Approssimazione fast-inv-sqrt | 0.000337 | 0.04266 |

### 5.4 VectorMagnitude

Calcolo della norma L2 di un vettore.

| Variante | Algoritmo | Energia Inv. (J) | Errore relativo |
|----------|-----------|-----------------|-----------------|
| `base` | NumPy `linalg.norm` | 0.000526 | 0.0 |
| `Light` | Approssimazione ridotta | 0.000421 | 0.04 |

### 5.5 SentimentAnalysis

Analisi del sentiment su testo, con modello NLP.

| Variante | Modello | Energia Inv. (J) | Qualità |
|----------|---------|-----------------|---------|
| `base` | Modello pesante (DistilBERT-like) | 0.091579 | high (0) |
| `Light` | Modello leggero (rule-based/VADER) | 0.072632 | low (2) |

### 5.6 ImageClassification

Classificazione di immagini con rete neurale convoluzionale.

| Variante | Modello | Energia Inv. (J) | Qualità |
|----------|---------|-----------------|---------|
| `base` | ResNet50 / EfficientNet | 1.455263 | high (0) |
| `Light` | MobileNet / SqueezeNet | < base | low (2) |

---

## 6. Sistema di Profilazione

### 6.1 Flusso di misurazione dell'energia

```
Invocazione → Esecuzione container → Lettura RAPL/perf → 
Write InfluxDB (measurement: energy_sample, tag: variant_id, 
field: invocation_joule) → EMA update in etcd
```

L'EMA (Exponential Moving Average) con α = 0.2 aggiorna la stima energetica
persistita in etcd dopo ogni invocazione:

$$E_{\text{new}} = \alpha \cdot E_{\text{measured}} + (1-\alpha) \cdot E_{\text{old}}$$

### 6.2 Script di profilazione (`ScriptTesi/profile_variants.py`)

Lo script automatizza il processo di raccolta delle statistiche energetiche:

```bash
python3 ScriptTesi/profile_variants.py \
    --function fibonacci-binet \
    --variant-id binet \
    --n-warmup 5 \
    --n-samples 30 \
    --param n:100 \
    --influx-url http://localhost:8086 \
    --influx-token <token> \
    --influx-org <org> \
    --influx-bucket <bucket>
```

**Fasi:**
1. **Warm-up**: `n_warmup` invocazioni non misurate (per inizializzare il container).
2. **Misurazione**: `n_samples` invocazioni con raccolta energie.
3. **Query InfluxDB**: calcolo di $\mu$, $\sigma$ sui campioni raccolti.
4. **Report**: stampa il frammento JSON da inserire nel file variante.

---

## 7. Design sperimentale

### 7.1 Funzioni e varianti da valutare

| Funzione | Varianti | Tipo output | Input tipico |
|----------|---------- |-------------|--------------|
| Fibonacci | 4 (base, opt-py, c, binet) | errore numerico | n=100 |
| PiLeibniz | 9 (base, n1000..n200000) | errore numerico | — |
| Sqrt | 2 (base, Light) | errore numerico | x=2.0 |
| VectorMagnitude | 2 (base, Light) | errore numerico | vec 1000-dim |
| SentimentAnalysis | 2 (base, Light) | qualità | testo 100 parole |
| ImageClassification | 2 (base, Light) | qualità | immagine 224×224 |

### 7.2 Esperimenti proposti

**Exp 1 — Comportamento di λ per zona geografica**

Per ogni zona (DE, ES, FR, NO, PL):
- Simulare CI dal P5 al P95 della distribuzione storica.
- Registrare quale variante viene selezionata ad ogni CI.
- Tracciare il grafico variante selezionata vs. CI.

**Exp 2 — Trade-off energia/accuratezza**

Per PiLeibniz (frontiera Pareto più ricca):
- Confrontare energia totale consumata (50 invocazioni) vs. errore cumulativo.
- Confrontare: sempre-base vs. sempre-n1000 vs. selezione-pareto-λ0.3 vs. λ0.7.

**Exp 3 — Convergenza UCB**

Con varianti Fibonacci:
1. Avviare con stime energetiche deliberatamente errate (+50%).
2. Eseguire 100 invocazioni con λ = 0.3 (rete sporca).
3. Tracciare l'evoluzione di LCB e della variante selezionata nel tempo.
4. Verificare convergenza verso la variante più efficiente.

**Exp 4 — Impatto della zona sul risparmio energetico**

Confrontare il consumo energetico totale di 1000 invocazioni Fibonacci in:
- Zona NO (pulita, quasi sempre λ ≈ 1): attesa selezione frequente di `c`
- Zona PL (sporca, quasi sempre λ ≈ 0): attesa selezione frequente di `binet`
- Zona DE (mixed): distribuzione mista

**Exp 5 — Sensibilità a β (exploration strength)**

Per Fibonacci/PiLeibniz con n_samples iniziale piccolo (3-5):
- Confrontare β ∈ {0, 0.5, 1.0, 2.0}.
- Misurare: quante volte viene esplorata la variante subottimale, e dopo quante
  invocazioni il sistema converge alla scelta ottimale.

### 7.3 Metriche di valutazione

| Metrica | Descrizione |
|---------|-------------|
| $E_{\text{tot}}$ | Energia totale consumata (J) in N invocazioni |
| $\varepsilon_{\text{med}}$ | Errore mediano dell'output |
| $n_{\text{exp}}$ | Numero di invocazioni "esplorative" (UCB attivo) |
| $t_{\text{conv}}$ | Invocazioni fino alla convergenza UCB |
| $\Delta E$ | Risparmio energetico vs. variante base (%) |

---

## 8. Conclusioni

Il sistema implementato integra tre componenti ortogonali che cooperano per una
schedulazione green-aware efficace:

1. **Sigmoide parametrizzata per zona**: λ riflette fedelmente il mix energetico locale,
   con parametri derivati da dati storici reali (ElectricityMaps). Non vi è alcun magic
   number globale: ogni zona ha la propria curva sigmoide calibrata.

2. **Frontiera di Pareto bi-obiettivo**: la selezione è strutturalmente corretta —
   non si seleziona mai una variante dominata. L'algoritmo O(n log n) è efficiente
   anche con molte varianti.

3. **Esplorazione UCB**: la selezione non è *statica* — apprende dai dati reali di
   produzione. Le stime energetiche offline sono punti di partenza; il sistema le
   raffina automaticamente nel tempo, garantendo che varianti potenzialmente ottimali
   ma poco campionate vengano esaminate prima di essere scartate.

La combinazione dei tre meccanismi realizza un allocatore che è allo stesso tempo
**corretto** (Pareto-ottimale), **context-aware** (λ dalla CI reale della zona), ed
**adattivo** (UCB che apprende dall'uso).
