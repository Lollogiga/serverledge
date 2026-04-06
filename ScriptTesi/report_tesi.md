# Report — Schedulazione Green-Aware di Funzioni Serverless tramite Selezione di Varianti Pareto-Ottimale con Esplorazione UCB

---

## 1. Motivazione

Il paradigma FaaS su infrastrutture edge introduce una tensione fondamentale tra due obiettivi
in conflitto:

1. **Precisione del risultato** — l'utente si aspetta output corretti o di alta qualità.
2. **Consumo energetico** — il nodo edge ha risorse limitate e opera in un contesto in cui
   il costo ambientale dell'energia varia nel tempo a seconda della quota di rinnovabili
   disponibile nella rete elettrica locale.

Per molte funzioni esistono *varianti approssimate* che producono risultati leggermente meno
precisi ma richiedono meno energia computazionale. Un sistema green-aware dovrebbe scegliere
automaticamente la variante più adeguata al contesto corrente: privilegiare l'accuratezza
quando la rete è alimentata da fonti rinnovabili, e risparmiare energia quando l'intensità
di carbonio è elevata.

Il presente lavoro estende Serverledge con tre meccanismi coordinati:

1. **Selezione di varianti Pareto-ottimale** guidata da un parametro λ derivato
   dall'intensità di carbonio della rete elettrica locale (API ElectricityMaps).
2. **Funzione sigmoide parametrizzata per zona geografica**, con parametri pre-calcolati
   offline da dati storici.
3. **Esplorazione UCB (Upper Confidence Bound)** che promuove la valutazione di varianti
   poco osservate, raffinando le stime energetiche con dati reali raccolti da InfluxDB.

---

## 2. Funzioni Utilizzate

Il sistema è stato valutato su sei funzioni raggruppate in due famiglie:

- **Funzioni numeriche** (PiLeibniz, LnHarmonic, MonteCarlo): output quantificabile,
  errore misurato come scarto relativo dal valore esatto.
- **Funzioni ML** (SentimentAnalysis, ImageClassification, SpamDetection):
  output di classificazione, qualità misurata come accuracy su benchmark di riferimento.

★ = variante approssimata sul fronte di Pareto.

---

### 2.1 PiLeibniz

Stima $\pi$ tramite la serie di Leibniz con $N$ termini:
$$\frac{\pi}{4} = \sum_{k=0}^{N-1} \frac{(-1)^k}{2k+1}$$
Maggiore è $N$, più accurata è la stima ma maggiore è il costo computazionale. È la funzione
con la frontiera di Pareto più ricca (nove varianti che coprono tre ordini di grandezza in
energia ed errore).

| Variante | N termini | Energia (J) | Errore rel. | Appross. |
|----------|:---------:|:-----------:|:-----------:|:--------:|
| `base` | adattivo | 0.007823 | 0.0 | No |
| `optimization-py` | adattivo | 0.007318 | 0.0 | No |
| `c` | adattivo | 0.001052 | 0.0 | No |
| `n200000` ★ | 200 000 | 0.001534 | 1.0×10⁻⁵ | Sì |
| `n100000` ★ | 100 000 | 0.001163 | 2.0×10⁻⁵ | Sì |
| `n50000` ★ | 50 000 | 0.000817 | 4.0×10⁻⁵ | Sì |
| `n10000` ★ | 10 000 | 0.000421 | 2.0×10⁻⁴ | Sì |
| `n5000` ★ | 5 000 | 0.000308 | 4.0×10⁻⁴ | Sì |
| `n1000` ★ | 1 000 | 0.000196 | 2.0×10⁻³ | Sì |

---

### 2.2 LnHarmonic

Stima $\ln(2)$ tramite la serie armonica alternante con $N$ termini:
$$\ln(2) = \sum_{k=1}^{N} \frac{(-1)^{k+1}}{k} = 1 - \frac{1}{2} + \frac{1}{3} - \frac{1}{4} + \ldots$$
Struttura analoga a PiLeibniz: il parametro $N$ bilancia accuratezza e consumo energetico.

| Variante | N termini | Energia (J) | Errore rel. | Appross. |
|----------|:---------:|:-----------:|:-----------:|:--------:|
| `base` | adattivo | 0.007700 | 0.0 | No |
| `n200000` ★ | 200 000 | 0.001534 | 3.6×10⁻⁶ | Sì |
| `n100000` ★ | 100 000 | 0.001163 | 7.2×10⁻⁶ | Sì |
| `n50000` ★ | 50 000 | 0.000817 | 2.9×10⁻⁵ | Sì |
| `n10000` ★ | 10 000 | 0.000421 | 1.4×10⁻⁴ | Sì |
| `n5000` ★ | 5 000 | 0.000308 | 2.9×10⁻⁴ | Sì |
| `n1000` ★ | 1 000 | 0.000196 | 1.4×10⁻³ | Sì |

---

### 2.3 MonteCarlo

Stima $\pi$ col metodo Monte Carlo: genera $N$ punti casuali in $[0,1]^2$ e conta quanti
cadono nel cerchio unitario. L'errore statistico cala come $O(1/\sqrt{N})$.
Tutte le varianti sono approssimate (l'algoritmo è intrinsecamente stocastico); il fronte di
Pareto copre l'intero set.

| Variante | N campioni | Energia (J) | Errore rel. | Appross. |
|----------|:----------:|:-----------:|:-----------:|:--------:|
| `n1000` ★ | 1 000 | 0.000147 | 0.0520 | Sì |
| `n5000` ★ | 5 000 | 0.000912 | 0.0232 | Sì |
| `n10000` ★ | 10 000 | 0.002023 | 0.0164 | Sì |
| `n50000` ★ | 50 000 | 0.014308 | 0.0074 | Sì |
| `n100000` ★ | 100 000 | 0.033710 | 0.0052 | Sì |
| `n500000` ★ | 500 000 | 0.231480 | 0.0023 | Sì |

---

### 2.4 SentimentAnalysis

Classifica un testo in input come **POSITIVE** o **NEGATIVE** (classificazione binaria).
Le varianti sono modelli Transformer di dimensioni decrescenti più un classificatore
lessicale (VADER/AFINN) come variante ultra-light. Qualità stimata su benchmark SST-2.

| Variante | Modello | Params | Energia (J) | Qualità | Appross. |
|----------|---------|:------:|:-----------:|:-------:|:--------:|
| `roberta-large` | siebert/sentiment-roberta-large | 355 M | 0.0916 | Alta | No |
| `distilbert` ★ | distilbert-base-uncased-sst-2 | 67 M | 0.0387 | Media | Sì |
| `bert-tiny` ★ | mrm8488/bert-tiny-sst2 | 4 M | 0.0113 | Media-bassa | Sì |
| `vader` ★ | AFINN lexicon (Python puro) | — | 0.0024 | Bassa | Sì |

---

### 2.5 ImageClassification

Classifica un'immagine in input (Base64) tra 1000 categorie ImageNet, restituendo la
categoria predetta e il confidence score. Le varianti sono architetture Transformer/CNN
di dimensioni decrescenti. Qualità indicata come Top-1 accuracy su ImageNet.

| Variante | Modello | Params | Energia (J) | Qualità | Appross. |
|----------|---------|:------:|:-----------:|:-------:|:--------:|
| `vit-base` | google/vit-base-patch16-224 | 86 M | 1.4553 | Alta | No |
| `efficientnet-b0` ★ | google/efficientnet-b0 | 5.3 M | 0.4180 | Media | Sì |
| `mobilenet-v2` ★ | google/mobilenet_v2_1.0_224 | 3.4 M | 0.2289 | Media-bassa | Sì |
| `mobilenet-v2-tiny` ★ | google/mobilenet_v2_0.35_224 | 1.7 M | 0.0770 | Bassa | Sì |

---

### 2.6 SpamDetection

Classifica un messaggio (email, SMS) come **SPAM** o **HAM** (classificazione binaria).
Le varianti spaziano da modelli Transformer fine-tuned per spam detection fino a un
classificatore euristico basato su keyword. Qualità stimata su corpus misti
(Enron + SMS Spam Collection + phishing samples).

| Variante | Modello | Params | Energia (J) | Qualità | Appross. |
|----------|---------|:------:|:-----------:|:-------:|:--------:|
| `roberta-spam` | mshenoda/roberta-spam | 125 M | 0.0850 | Alta | No |
| `distilbert` ★ | Falconsai/spam_classification | 67 M | 0.0350 | Media | Sì |
| `bert-tiny` ★ | mrm8488/bert-tiny-finetuned-sms-spam | 4.4 M | 0.0100 | Media-bassa | Sì |
| `keyword` ★ | Keyword heuristic (Python puro) | — | 0.0020 | Bassa | Sì |

---

## 3. Esplorazione UCB

### 3.1 Motivazione

Il profilo energetico offline di ogni variante (`invocation_joule` nel JSON di
configurazione) è ottenuto eseguendo ripetutamente le varianti su un hardware di
riferimento e registrando l'energia media consumata.
Questo approccio ha due limitazioni concrete che hanno reso necessario il meccanismo
di esplorazione:

1. **Dipendenza dall'hardware**: i profili energetici misurati sull'hardware di sviluppo
   non corrispondono necessariamente ai consumi reali su un nodo edge con CPU o workload
   diversi. I valori offline potrebbero sovra- o sotto-stimare sistematicamente il
   consumo reale, portando lo scheduler a preferire stabilmente una variante nominalmente
   "economica" che in produzione è invece costosa — e viceversa.

2. **Scalabilità della profilazione**: per ogni nuova variante sarebbe necessario
   ripetere una profilazione offline esaustiva prima di poterla deployare. Questo crea
   un collo di bottiglia nello sviluppo e impedisce l'aggiunta rapida di nuove varianti
   al sistema.

Indipendentemente da queste due limitazioni principali, vale la pena notare che il
consumo energetico non è stazionario — dipende dal carico del sistema, dalla temperatura
e dallo stato della cache del container. Una stima offline non cattura questa variabilità.

Il meccanismo UCB (Upper Confidence Bound) affronta queste problematiche con il principio
di **ottimismo sotto incertezza** (Optimism in the Face of Uncertainty): per varianti
poco osservate, il sistema assume *ottimisticamente* che il loro consumo energetico
reale potrebbe essere inferiore alla stima corrente. Questo incentiva l'esplorazione
di varianti sotto-campionate, raccogliendo dati reali da InfluxDB e convergendo
progressivamente a stime accurate senza richiedere alcuna profilazione offline preventiva.

### 3.2 Come vengono acquisiti i dati

Il flusso di acquisizione è il seguente:

```
Invocazione
    │
    ▼
Esecuzione container
    │
    ▼
Lettura RAPL / perf-stat  (energia CPU durante l'esecuzione)
    │
    ▼
Write su InfluxDB
    measurement : energy_sample
    tag         : variant_id, function_name
    fields      : invocation_joule, cold_start_joule, timestamp
    │
    ▼
EMA update su etcd  (α = 0.2)
    E_new = 0.2 · E_measured + 0.8 · E_old
```

Ogni invocazione produce quindi due effetti persistenti:

- **InfluxDB** accumula la serie storica delle misurazioni per variante; da essa si
  estraggono σ e n utilizzati nel termine di esplorazione UCB.
- **etcd** mantiene la stima EMA corrente: è il valore μ primario nella formula LCB,
  nonché il fallback unico quando InfluxDB non è disponibile o ha campioni insufficienti.

#### Formula LCB (Lower Confidence Bound)

Per la *minimizzazione* energetica si usa il limite inferiore di confidenza (equivalente
all'UCB classico applicato alla quantità negata $-E$):

$$\text{LCB}_i = \mu^{\text{EMA}}_i - \beta \cdot \frac{\sigma^{\text{influx}}_i}{\sqrt{n_i}}$$

dove:

| Simbolo | Significato |
|---------|-------------|
| $\mu^{\text{EMA}}_i$ | Stima EMA dell'energia della variante $i$ (da etcd, α = 0.2) |
| $\sigma^{\text{influx}}_i$ | Deviazione standard calcolata sui campioni InfluxDB (ultimi 24 h) |
| $n_i$ | Numero di campioni InfluxDB disponibili |
| $\beta$ | Parametro di esplorazione (configurabile, default 1.0) |

**Perché EMA come μ?** L'EMA è un estimatore ricorrente con fattore di smorzamento
α = 0.2: pondera di più le misurazioni recenti e dimentica progressivamente quelle
vecchie. Questo lo rende più robusto rispetto a una media aritmetica su 24 h quando
il consumo energetico del nodo varia nel tempo (temperatura, carico CPU, stato cache).

**Proprietà:**
- $n_i$ grande (variante ben esplorata): $\sigma_i / \sqrt{n_i} \to 0$ → $\text{LCB}_i \to \mu^{\text{EMA}}_i$ (sfruttamento)
- $n_i$ piccolo (variante sotto-esplorata): correzione ampia → $\text{LCB}_i \ll \mu^{\text{EMA}}_i$ (esplorazione ottimistica)
- $\beta = 0$: UCB disabilitato, si usa solo l'EMA
- $\beta \to \infty$: massima esplorazione

**Fallback:** se InfluxDB non è raggiungibile, oppure la variante ha meno di
$\texttt{min\_samples}$ (default: 5) campioni — soglia sotto la quale σ sarebbe
inaffidabile — lo scheduler usa l'EMA da etcd senza correzione.

## 4. Scheduling Green-Aware

### 4.1 Visione ad alto livello

Quando arriva una richiesta di invocazione per una funzione logica (es. `SpamDetection`),
lo scheduler esegue i seguenti passi:

```
1. Recupera la lista di varianti fisiche dal JSON (etcd)
2. Per ogni variante: ottieni energia stimata
       → recupera µ_EMA da etcd
       → se UCB attivo e n_i ≥ min_samples: LCB = µ_EMA - β·σ_influx/√n
       → altrimenti: usa µ_EMA direttamente (nessuna correzione)
3. Calcola la frontiera di Pareto bi-obiettivo (energia ↓, errore ↓)
4. Calcola λ dalla carbon intensity corrente della zona del nodo
5. Scalarizza la frontiera con λ → seleziona la variante con score minimo
6. Invoca la variante selezionata; registra l'energia misurata
```

L'obiettivo è scegliere la variante che realizza il miglior compromesso tra qualità del
risultato e costo energetico, nel contesto energetico corrente del nodo edge.

### 4.2 Il parametro λ

#### Frontiera di Pareto

Dati $n$ varianti, il sistema calcola la frontiera di Pareto bi-obiettivo
(energia, errore), entrambi da minimizzare:

1. Ordina le varianti per energia crescente; a parità, per errore crescente.
2. Scansione singola: una variante entra nella frontiera se e solo se il suo errore è
   strettamente minore del minimo errore dei punti precedenti.

L'algoritmo è O(n log n) totale e identifica tutti e soli i punti non dominati:
nessuna variante nella frontiera è "peggiore su entrambi gli assi" rispetto a un'altra.

**Esempio (PiLeibniz, varianti selezionate):**

| Variante | Energia (J) | Errore | Pareto? |
|----------|:-----------:|:------:|:-------:|
| `n1000` | 0.000196 | 2.0×10⁻³ | ✓ |
| `n5000` | 0.000308 | 4.0×10⁻⁴ | ✓ |
| `n10000` | 0.000421 | 2.0×10⁻⁴ | ✓ |
| `n50000` | 0.000817 | 4.0×10⁻⁵ | ✓ |
| `n100000` | 0.001163 | 2.0×10⁻⁵ | ✓ |
| `n200000` | 0.001534 | 1.0×10⁻⁵ | ✓ |
| `c` | 0.001052 | 0.0 | ✓ |
| `optimization-py` | 0.007318 | 0.0 | ✗ dominata da `c` |
| `base` | 0.007823 | 0.0 | ✗ dominata da `c` |

#### Scalarizzazione con λ

Data la frontiera di Pareto, ogni variante $i$ riceve un punteggio:

$$\text{score}(i) = (1 - \lambda) \cdot \hat{E}_i + \lambda \cdot \hat{\varepsilon}_i$$

dove $\hat{E}_i$ e $\hat{\varepsilon}_i$ sono i valori di energia ed errore normalizzati
nell'intervallo $[0,1]$ rispetto al range della frontiera. La variante con il
**punteggio minimo** viene selezionata.

**Cosa rappresenta λ:**

$\lambda \in [0, 1]$ è un *peso di qualità* che esprime quanto privilegiare
l'accuratezza del risultato rispetto all'efficienza energetica nell'istante corrente.

- **λ → 0** (rete sporca, alta CI): il termine dominante è $(1-\lambda)\hat{E}_i$ →
  il sistema seleziona la variante più efficiente energeticamente.
- **λ → 1** (rete pulita, bassa CI): il termine dominante è $\lambda\hat{\varepsilon}_i$ →
  il sistema seleziona la variante più accurata (errore minimo).
- **λ = 0.5**: selezione al "ginocchio" della frontiera, bilanciando i due obiettivi.

λ non è un parametro configurato manualmente: viene calcolato automaticamente in tempo
reale dalla carbon intensity della zona geografica del nodo edge.

### 4.3 Calcolo di λ dalla Carbon Intensity

La **carbon intensity** (CI, misurata in gCO₂eq/kWh) quantifica quante emissioni di
CO₂ equivalente vengono prodotte per ogni kWh di energia consumata dalla rete elettrica
locale. Viene ottenuta in tempo reale dall'API ElectricityMaps (con cache di 15 minuti).

λ è derivato tramite una **funzione sigmoide logistica** centrata sui valori storici della zona:

$$\lambda = \frac{1}{1 + \exp\!\left(k \cdot \dfrac{CI - \mu_z}{\sigma_z}\right)}$$

dove:

| Simbolo | Significato |
|---------|-------------|
| $CI$ | Carbon intensity corrente (gCO₂eq/kWh) |
| $\mu_z$ | Media storica della CI per la zona $z$ |
| $\sigma_z$ | Parametro di scala: $\sigma_z = (P_{95} - P_5) / (2 \ln\frac{1-\alpha}{\alpha})$, con $\alpha=0.1$ |
| $k$ | Pendenza della sigmoide (default 0.45) |

La formula per $\sigma_z$ assicura che $\lambda(P_{95}) = \alpha = 0.1$ e
$\lambda(P_5) = 1 - \alpha = 0.9$: ai valori estremi della distribuzione storica, λ
raggiunge i valori estremi dell'intervallo, massimizzando la sensibilità del sistema
nelle condizioni reali attese per quella zona.

### 4.4 Dipendenza dalla zona geografica

La sigmoide è **parametrizzata per zona** perché la distribuzione storica della CI varia
enormemente tra Paesi con mix energetici diversi.

Se si usasse una sigmoide globale con μ unica, sarebbe completamente insensibile alle
variazioni locali: in Francia (CI media ≈ 14 gCO₂/kWh), anche i picchi locali non
superano mai la media globale e λ sarebbe sempre vicino a 1 indipendentemente dal
contesto. In Polonia (CI media ≈ 499 gCO₂/kWh), λ sarebbe sempre vicino a 0.

Usando μ e σ calibrati sulla distribuzione storica *di quella zona*, la sigmoide
risponde alle oscillazioni locali: quando la CI di una zona è sopra la sua media
storica, λ scende (si risparmia energia); quando è sotto la media, λ sale (si usa
la variante più precisa). Il sistema si adatta all'energia disponibile nel preciso
contesto geografico del nodo, non a una scala globale arbitraria.

I parametri sono pre-calcolati offline da dati storici scaricati da ElectricityMaps e
salvati in `zone_ci_params.csv`. Le cinque zone scelte coprono l'intero spettro della
carbon intensity europea — da una rete quasi completamente idroelettrica (Norvegia) o
nucleare (Francia) a una rete a carbone dominante (Polonia) — garantendo che il sistema
venga validato su regimi di λ bassi, medi e alti:

| Zona | μ (gCO₂/kWh) | σ | Caratteristica rete |
|------|:------------:|:-:|---------------------|
| NO | 6 | ~1 | Idroelettrico dominante → quasi sempre λ ≈ 1 |
| FR | 14 | ~4 | Nucleare dominante → λ solitamente alto |
| ES | 95 | ~14 | Mix rinnovabile/gas → λ variabile |
| DE | 278 | ~41 | Mix carbone/rinnovabili → λ mediamente basso |
| PL | 499 | ~35 | Carbone dominante → quasi sempre λ ≈ 0 |

---

## 5. Bozza degli Esperimenti

Oltre agli esperimenti descritti di seguito, la sperimentazione include alcune analisi
preliminari che ne costituiscono il presupposto: (i) la profilazione offline del consumo
energetico di ogni variante tramite invocazioni ripetute con lettura da InfluxDB; (ii) la
calibrazione della sigmoide per zona geografica a partire dai dati storici di carbon
intensity scaricati da ElectricityMaps.

### 5.1 Exp 1 — Frontiera di Pareto e selezione per λ

**Obiettivo:** per ogni famiglia di funzioni, visualizzare la frontiera di Pareto
(energia vs. errore/qualità) e mostrare quale variante viene selezionata al variare di
λ, evidenziando come la scalarizzazione si sposta lungo la frontiera in funzione del
peso attribuito all'efficienza energetica rispetto all'accuratezza.

**Metodologia:**
- Per ogni funzione, calcolare la frontiera di Pareto bi-obiettivo (energia, errore).
- Spazzare λ ∈ [0, 1] a passi di 0.1 e registrare quale variante viene selezionata
  dallo scheduler in assenza di rumore (stime energetiche = valori offline).
- Visualizzare: scatter plot energia vs. errore con fronte evidenziato, e un grafico
  "variante selezionata vs. λ" che mostra la transizione lungo il fronte.

**Risultato atteso:** la variante selezionata si sposta monotonicamente dalla più
efficiente (λ = 0) alla più accurata (λ = 1); le transizioni si collocano ai valori di
λ in cui lo score delle varianti si incrociano.

---

### 5.2 Exp 2 — Comportamento dello scheduling al variare della CI

**Obiettivo:** verificare che il sistema selezioni varianti diverse in funzione della
carbon intensity della zona, con una transizione continua dalla variante più accurata
(CI bassa) a quella più efficiente (CI alta) che rispecchi il comportamento atteso
della sigmoide calibrata per zona.

**Metodologia:**
- Per ogni zona (NO, FR, ES, DE, PL) e per funzioni rappresentative di entrambe le
  famiglie (es. `PiLeibniz`, `SpamDetection`):
  - Invocare lo scheduler con la CI storica mediana di quella zona.
  - Registrare la variante selezionata e il valore di λ.
- Mettere in risalto il rapporto fra accuratezza del risultato e consumo energetico
  in funzione dei valori di CI.

**Risultato atteso:**
- NO (CI ≈ 6) → λ ≈ 1 → variante più accurata
- FR (CI ≈ 14) → λ alto → variante heavy/medium
- ES (CI ≈ 95) → λ medio → variante di compromesso
- DE (CI ≈ 278) → λ basso → variante leggera
- PL (CI ≈ 499) → λ ≈ 0 → variante più efficiente

---

### 5.3 Exp 3 — Risparmio energetico vs. baseline always-accurate

**Obiettivo:** quantificare il risparmio energetico dello scheduler green-aware (in
termini di CI) rispetto a uno scheduler che seleziona sempre la variante più accurata,
su un workload distribuito su più zone con carbon intensity diverse.

**Metodologia:**
- Workload: invocazioni distribuite su tutte e 6 le funzioni, con distribuzione uniforme
  delle zone (NO, FR, ES, DE, PL).
- Strategie a confronto:
  1. Baseline always-accurate: seleziona sempre la variante con `is_approximate: false`
     più precisa, indipendentemente dalla CI.
  2. Scheduler green-aware (questo lavoro).
- Metriche: energia totale (J), risparmio percentuale $\Delta E\%$, errore/accuracy medio.

**Risultato atteso:** lo scheduler produce un risparmio energetico significativo rispetto
alla baseline, in particolare nelle zone con CI elevata (DE, PL), mantenendo qualità
alta nelle zone pulite (NO, FR).

---

### 5.4 Exp 4 — Convergenza EMA con stime iniziali errate

**Obiettivo:** verificare che il meccanismo EMA+LCB corregga autonomamente stime
energetiche iniziali errate attraverso le misurazioni reali, mettendo in risalto la
prima fase di esplorazione.

**Metodologia:**
- Funzione: `PiLeibniz`.
- Setup: alterare artificialmente `invocation_joule` di una variante approssimata nel
  JSON portandolo a un valore significativamente gonfiato rispetto al reale.
- Forzare un numero iniziale di campioni InfluxDB per attivare il termine LCB, quindi
  eseguire 100 invocazioni di scheduling con λ basso (zona DE — preferisce efficienza),
  registrando ad ogni step: variante selezionata, µ_EMA, LCB della variante perturbata.

**Come mostrare la convergenza:**
Tracciare su un unico grafico (asse X = invocazioni della variante perturbata):
- µ_EMA che decae esponenzialmente dal valore gonfiato verso il valore reale
- Soglia corrispondente alla variante concorrente — quando µ_EMA scende sotto questa
  soglia, la variante perturbata diventa di nuovo competitiva
- Punti colorati sulla timeline: quale variante viene selezionata dallo scheduler

La convergenza è descrivibile analiticamente: con α = 0.2, valore reale $E^*$, valore
iniziale $E_0$:
$$\mu^{\text{EMA}}(t) = E^* + (E_0 - E^*) \cdot (1-\alpha)^t$$

**Risultato atteso:** il meccanismo LCB rende la variante perturbata attrattiva in fase
esplorativa; man mano che si accumulano misurazioni reali, l'EMA converge al valore
reale e la selezione diventa stabile e corretta.

---

### 5.5 Exp 5 — Sensibilità al parametro β

**Obiettivo:** β = 0 corrisponde al puro sfruttamento (nessuna esplorazione); valori
maggiori aumentano la robustezza a stime iniziali errate a scapito di qualche invocazione
esplorativa in più. Questo esperimento confronta questi regimi.

**Metodologia:**
- Funzione: `PiLeibniz` (fronte di Pareto ricco, 9 varianti).
- Setup: stime energetiche iniziali volutamente imprecise (pochi campioni offline).
- β ∈ {0, 0.5, 1.0, 2.0}.
- Per ciascun β: 200 invocazioni, registrando variante selezionata ad ogni step,
  numero di invocazioni su varianti sub-ottimali, e invocazione di convergenza.

**Risultato atteso:**
- β = 0: puro sfruttamento — se la stima iniziale è errata, non converge mai.
- β = 0.5–1.0: convergenza in 15–30 invocazioni con overhead esplorativo limitato.
- β = 2.0: esplora molto di più, convergenza più lenta ma più robusta a outlier.

---

### 5.6 Metriche di valutazione

| Metrica | Descrizione |
|---------|-------------|
| $E_{\text{tot}}$ | Energia totale consumata (J) in $N$ invocazioni |
| $\varepsilon_{\text{med}}$ | Errore relativo mediano dell'output (funzioni numeriche) |
| $A_{\text{med}}$ | Accuracy media delle classificazioni (funzioni ML) |
| $\Delta E\%$ | Risparmio energetico vs. baseline always-accurate: $(E_\text{base} - E_\text{sched})/E_\text{base}$ |
| $n_{\text{exp}}$ | Numero di invocazioni esplorative (UCB attivo, $n_i < \texttt{min\_samples}$) |
| $t_{\text{conv}}$ | Invocazioni fino alla convergenza UCB (variante ottimale stabile) |
| $\lambda_{\text{med}}$ | Valore mediano di λ per zona (verifica calibrazione sigmoide) |
