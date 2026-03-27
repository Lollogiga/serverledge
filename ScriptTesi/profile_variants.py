#!/usr/bin/env python3
"""
profile_variants.py — Profiling subsystem for Serverledge variant energy measurement.

Measures real invocation energy for one variant (or all variants in a JSON file)
by running N invocations, reading the energy statistics from InfluxDB, and
optionally writing the measured value back into the variant JSON file.

─── Single-variant mode ───────────────────────────────────────────────────────
    python3 profile_variants.py \
        --function fibonacci-c \
        --variant-id c \
        --n-samples 30 \
        --param n:100 \
        --update-json ../variants/fibonacci/fibonacci.json

─── Batch mode (all variants in a JSON) ───────────────────────────────────────
    python3 profile_variants.py \
        --batch-json ../variants/fibonacci/fibonacci.json \
        --function-prefix fibonacci \
        --n-samples 30 \
        --param n:100

    In batch mode: each variant's function name is   <function-prefix>-<variant-id>

─── Environment variables (fallback for InfluxDB credentials) ─────────────────
    INFLUX_URL, INFLUX_TOKEN, INFLUX_ORG, INFLUX_BUCKET

Requirements:
    pip install influxdb-client
"""

import argparse
import json
import math
import os
import subprocess
import sys
import time

# InfluxDB client (optional; graceful degradation if not installed)
try:
    from influxdb_client import InfluxDBClient
    HAS_INFLUX = True
except ImportError:
    HAS_INFLUX = False
    print("[WARN] influxdb-client not installed; InfluxDB stats will be skipped.", file=sys.stderr)
    print("       Install with: pip install influxdb-client", file=sys.stderr)


# ---------------------------------------------------------------------------
# CLI helpers
# ---------------------------------------------------------------------------

def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(
        description="Profile a Serverledge function variant and report/update energy statistics."
    )
    # ── single-variant args ──
    p.add_argument("--function", default=None,
                   help="Function name as registered in Serverledge")
    p.add_argument("--variant-id", default=None,
                   help="Variant identifier (e.g. 'c', 'binet')")
    p.add_argument("--update-json", default=None, metavar="PATH",
                   help="Path to the variant JSON file. When set, writes the measured "
                        "invocation_joule back into the file automatically.")
    # ── batch args ──
    p.add_argument("--batch-json", default=None, metavar="PATH",
                   help="Path to a variant JSON file. Profiles ALL variants listed in it.")
    p.add_argument("--function-prefix", default=None,
                   help="Batch mode: function name = <prefix>-<variant-id>")
    # ── common args ──
    p.add_argument("--n-warmup", type=int, default=3,
                   help="Number of warm-up invocations (not measured)")
    p.add_argument("--n-samples", type=int, default=30,
                   help="Number of measured invocations")
    p.add_argument("--param", action="append", default=[],
                   metavar="KEY:VALUE", help="Function parameters (repeatable)")
    p.add_argument("--serverledge-cli", default="../../bin/serverledge-cli",
                   help="Path to serverledge-cli binary")
    p.add_argument("--influx-url",    default=None)
    p.add_argument("--influx-token",  default=None)
    p.add_argument("--influx-org",    default=None)
    p.add_argument("--influx-bucket", default=None)
    p.add_argument("--window", default="1h",
                   help="InfluxDB lookback window (e.g. '1h', '24h')")
    p.add_argument("--cold-start", action="store_true",
                   help="Measure cold-start energy (re-creates container each time)")
    p.add_argument("--wait-after", type=int, default=60,
                   help="Max seconds to wait for InfluxDB ingestion after invocations (default: 60)")
    p.add_argument("--poll-interval", type=int, default=5,
                   help="Seconds between InfluxDB polling attempts (default: 5)")
    return p


def invoke_function(cli: str, function: str, params: list[str], cold: bool) -> bool:
    """Invoke function via serverledge-cli. Returns True on success."""
    cmd = [cli, "invoke", "--function", function]
    for p in params:
        cmd += ["--param", p]
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=60)
        if result.returncode != 0:
            print(f"  [WARN] invocation failed: {result.stderr.strip()}", file=sys.stderr)
            return False
        return True
    except subprocess.TimeoutExpired:
        print("  [WARN] invocation timed out", file=sys.stderr)
        return False


# ---------------------------------------------------------------------------
# InfluxDB statistics
# ---------------------------------------------------------------------------

def query_energy_stats(url: str, token: str, org: str, bucket: str,
                       container_id: str, window: str) -> dict | None:
    """Query InfluxDB for energy statistics of a variant.

    Filters by function_name tag (e.g. 'montecarlo-n1000'),
    which is how serverledge tags measurements in InfluxDB.
    Returns dict or None.
    """
    if not HAS_INFLUX:
        return None

    client = InfluxDBClient(url=url, token=token, org=org)
    query_api = client.query_api()

    flux = f"""
from(bucket: "{bucket}")
  |> range(start: -{window})
  |> filter(fn: (r) => r._measurement == "energy_sample")
  |> filter(fn: (r) => r.function_name == "{container_id}")
  |> filter(fn: (r) => r._field == "invocation_joule")
  |> group()
  |> reduce(
       identity: {{n: 0, sum: 0.0, sum2: 0.0}},
       fn: (r, accumulator) => ({{
         n:    accumulator.n    + 1,
         sum:  accumulator.sum  + r._value,
         sum2: accumulator.sum2 + r._value * r._value
       }})
  )
"""
    try:
        tables = query_api.query(flux)
        for table in tables:
            for record in table.records:
                values = record.values
                n    = int(values.get("n", 0))
                s    = float(values.get("sum", 0.0))
                s2   = float(values.get("sum2", 0.0))
                if n > 0:
                    mean = s / n
                    var  = max(s2 / n - mean * mean, 0.0)
                    return {
                        "n":      n,
                        "mean":   mean,
                        "stddev": math.sqrt(var),
                        "min":    None,  # not computed in this pass
                        "max":    None,
                    }
    except Exception as exc:
        print(f"[WARN] InfluxDB query failed: {exc}", file=sys.stderr)
    finally:
        client.close()
    return None

# ---------------------------------------------------------------------------
# JSON update helper
# ---------------------------------------------------------------------------

def update_json_energy(json_path: str, variant_id: str, new_joule: float) -> bool:
    """
    Write the measured invocation_joule back into the variant JSON file.

    Finds the variant by id, updates energy.invocation_joule, sets
    energy.measured = true, and saves the file in-place.

    Returns True on success.
    """
    try:
        with open(json_path) as f:
            data = json.load(f)
    except Exception as exc:
        print(f"[ERROR] Cannot read {json_path}: {exc}", file=sys.stderr)
        return False

    found = False
    for variant in data.get("variants", []):
        if variant.get("id") == variant_id:
            variant["energy"]["invocation_joule"] = round(new_joule, 9)
            variant["energy"]["measured"] = True   # flag: value comes from real measurement
            found = True
            break

    if not found:
        print(f"[WARN] variant_id '{variant_id}' not found in {json_path}")
        return False

    try:
        with open(json_path, "w") as f:
            json.dump(data, f, indent=2)
            f.write("\n")
        print(f"  [JSON] Updated {json_path}: variant={variant_id} "
              f"invocation_joule={new_joule:.9f}")
        return True
    except Exception as exc:
        print(f"[ERROR] Cannot write {json_path}: {exc}", file=sys.stderr)
        return False


# ---------------------------------------------------------------------------
# Core: profile one variant
# ---------------------------------------------------------------------------

def profile_one(
    cli: str,
    function: str,
    variant_id: str,
    params: list,
    n_warmup: int,
    n_samples: int,
    cold_start: bool,
    influx_url: str | None,
    influx_token: str | None,
    influx_org: str | None,
    influx_bucket: str | None,
    window: str,
    update_json_path: str | None,
    wait_after: int = 60,
    poll_interval: int = 5,
) -> float | None:
    """
    Run warm-up + measured invocations for one variant.
    Returns measured mean invocation energy (J), or None if unavailable.
    """
    print(f"\n{'='*60}")
    print(f"  Profiling: {function}  (variant-id: {variant_id})")
    print(f"  Warm-up: {n_warmup}   Samples: {n_samples}")
    if params:
        print(f"  Params:  {params}")
    print(f"{'='*60}\n")

    # Snapshot current sample count BEFORE invocations (to detect new arrivals later)
    n_before = 0
    if all([influx_url, influx_token, influx_org, influx_bucket]):
        stats_pre = query_energy_stats(influx_url, influx_token, influx_org, influx_bucket,
                                       function, window)
        n_before = stats_pre["n"] if stats_pre else 0

    # Warm-up
    print(f"[Step 1/3] Warm-up ({n_warmup} invocations)...")
    for i in range(n_warmup):
        invoke_function(cli, function, params, cold=False)
        print(f"  [{i+1}/{n_warmup}]", end="\r")
    print(f"  Warm-up complete.{' ' * 20}")

    # Measured invocations
    print(f"\n[Step 2/3] Measuring ({n_samples} invocations)...")
    success = 0
    t0 = time.time()
    for i in range(n_samples):
        ok = invoke_function(cli, function, params, cold=cold_start)
        if ok:
            success += 1
        percent = int(100 * (i + 1) / n_samples)
        bar = "█" * (percent // 5) + "░" * (20 - percent // 5)
        print(f"  [{bar}] {percent}%  ({i+1}/{n_samples})", end="\r")
    elapsed = time.time() - t0
    print(f"  {success}/{n_samples} successful  ({elapsed:.1f}s){' ' * 20}\n")

    # Wait for InfluxDB ingestion with polling.
    # Kepler batches energy over its scrape interval (~15-60s), so N fast invocations
    # produce far fewer than N InfluxDB records. We therefore query the count BEFORE
    # invocations start (n_before) and exit as soon as any NEW record appears (n > n_before).
    stats = None
    if all([influx_url, influx_token, influx_org, influx_bucket]):
        print(f"[Step 3/3] Waiting for InfluxDB ingestion (max {wait_after}s, poll every {poll_interval}s)...")
        print(f"  Baseline: {n_before} existing samples in window '{window}'")
        deadline = time.time() + wait_after
        attempt = 0
        while time.time() < deadline:
            attempt += 1
            time.sleep(poll_interval)
            elapsed_w = int(time.time() - (deadline - wait_after))
            stats = query_energy_stats(influx_url, influx_token, influx_org, influx_bucket,
                                       function, window)
            n_got = stats["n"] if stats else 0
            new_samples = n_got - n_before
            print(f"  poll {attempt}: {n_got} total ({new_samples:+d} new) "
                  f"({elapsed_w}s/{wait_after}s)", end="\r")
            if new_samples > 0:
                print(f"  poll {attempt}: {n_got} total ({new_samples:+d} new) — ingestion complete.{' '*10}")
                break
        else:
            n_got = stats["n"] if stats else 0
            print(f"\n  [WARN] Timeout after {wait_after}s: no new samples in InfluxDB.")
            print("         Increase --wait-after if Kepler/Prometheus scrape interval is slow.")
    else:
        print("[WARN] InfluxDB credentials not configured — skipping stats query.")

    # Report
    print("\n" + "─" * 60)
    print(f"  Function   : {function}  |  Variant: {variant_id}")
    print(f"  Samples    : {success}/{n_samples} succeeded")
    if stats and stats["n"] > 0:
        cv = (100 * stats["stddev"] / stats["mean"]) if stats["mean"] > 0 else float("nan")
        print(f"  n={stats['n']}  mean={stats['mean']:.8f} J  "
              f"σ={stats['stddev']:.8f} J  CV={cv:.1f}%")
        print(f"  → invocation_joule = {stats['mean']:.8f}")
        if update_json_path:
            update_json_energy(update_json_path, variant_id, stats["mean"])
        return stats["mean"]
    else:
        print("  [WARN] No InfluxDB stats — cannot update JSON.")
        return None


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    parser = build_parser()
    args = parser.parse_args()

    # Resolve InfluxDB connection params (CLI args > env vars)
    influx_url    = args.influx_url    or os.environ.get("INFLUX_URL")
    influx_token  = args.influx_token  or os.environ.get("INFLUX_TOKEN")
    influx_org    = args.influx_org    or os.environ.get("INFLUX_ORG")
    influx_bucket = args.influx_bucket or os.environ.get("INFLUX_BUCKET")

    cli = args.serverledge_cli
    if not os.path.isfile(cli):
        print(f"[ERROR] serverledge-cli not found at: {cli}", file=sys.stderr)
        sys.exit(1)

    common = dict(
        n_warmup=args.n_warmup,
        n_samples=args.n_samples,
        params=args.param,
        cold_start=args.cold_start,
        influx_url=influx_url, influx_token=influx_token,
        influx_org=influx_org, influx_bucket=influx_bucket,
        window=args.window,
        wait_after=args.wait_after,
        poll_interval=args.poll_interval,
    )

    # ── Batch mode ──────────────────────────────────────────────────────────
    if args.batch_json:
        if not args.function_prefix:
            print("[ERROR] --batch-json requires --function-prefix", file=sys.stderr)
            sys.exit(1)
        try:
            with open(args.batch_json) as f:
                data = json.load(f)
        except Exception as exc:
            print(f"[ERROR] Cannot read {args.batch_json}: {exc}", file=sys.stderr)
            sys.exit(1)

        variants = data.get("variants", [])
        print(f"\nBatch profiling {len(variants)} variant(s) from {args.batch_json}")
        results = {}
        for v in variants:
            vid  = v["id"]
            fname = f"{args.function_prefix}-{vid}"
            measured = profile_one(
                cli=cli, function=fname, variant_id=vid,
                update_json_path=args.batch_json,
                **common,
            )
            results[vid] = measured

        print("\n" + "=" * 60)
        print("  BATCH SUMMARY")
        print("=" * 60)
        for vid, val in results.items():
            status = f"{val:.8f} J" if val is not None else "N/A"
            print(f"  {vid:<25} {status}")
        print("=" * 60 + "\n")
        return

    # ── Single-variant mode ──────────────────────────────────────────────────
    if not args.function or not args.variant_id:
        print("[ERROR] Provide either --batch-json OR both --function and --variant-id",
              file=sys.stderr)
        sys.exit(1)

    profile_one(
        cli=cli,
        function=args.function,
        variant_id=args.variant_id,
        update_json_path=args.update_json,
        **common,
    )


if __name__ == "__main__":
    main()
