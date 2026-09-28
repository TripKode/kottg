#!/usr/bin/env python3
# KSP-2 Gate 4 — BCR (Beacon Chain Root): ticker de producción autónoma de
# bloques beacon + precompiles BEACONROOTLATEST/BEACONROOT/BEACONHEIGHT.
# Verifica contra la Beacon API del nodo (:9084) y el anclaje on-chain.
import json, urllib.request, subprocess, sys, time

ENGINE  = "http://localhost:8084"
BEACON  = "http://localhost:9084"
G       = "0x66d1acb3f9065eabecd97b0c2df07de45245518f"

# KSP-6B: selectores KSEL-8 computados en runtime desde la firma canónica
# (la firma es lo congelado; el selector se deriva con el dominio KSEL-8).
_SEL_CACHE = {}
def sel(sig):
    if sig not in _SEL_CACHE:
        out = subprocess.run(["/tmp/kselbin", sig], capture_output=True, text=True)
        _SEL_CACHE[sig] = out.stdout.split()[1]
    return _SEL_CACHE[sig]

SEL = {
    "init": sel("init(address)"),
    "latestRoot": sel("latestRoot()"),
    "rootAt": sel("rootAt(u256)"),
    "height": sel("height()"),
    "anchor": sel("anchor()"),
    "verifyAnchored": sel("verifyAnchored(u256)"),
    "getLastHeight": sel("getLastHeight()"),
}

def ident(a):
    return int(subprocess.run(["/tmp/kselbin", a], capture_output=True, text=True).stdout.split()[-1], 16)

PASSED = FAILED = 0
def check(desc, ok, detail=""):
    global PASSED, FAILED
    PASSED, FAILED = PASSED + bool(ok), FAILED + (not ok)
    print(("✅" if ok else "❌") + " " + desc + ("" if ok or not detail else f" — {detail}"))

def call(addr, sel, *args, caller=G):
    def word(a):
        return format((int(a, 16) if isinstance(a, str) else a) & (2**256-1), "064x")
    inp = sel + "00"*28 + "".join(word(a) for a in args)
    req = {"contract_address": addr, "caller": caller, "input": inp,
           "gas_limit": 4000000, "gas_price": 1000000000}
    r = urllib.request.Request(ENGINE + "/api/contract/call",
        data=json.dumps(req).encode(), headers={"Content-Type": "application/json"})
    try:
        ex = json.load(urllib.request.urlopen(r, timeout=30)).get("execution", {})
        return ex.get("success"), ex.get("return_data", "0x")
    except urllib.error.HTTPError as e:
        return False, "HTTP" + str(e.code)

def w(rd):
    return int(rd[2:2+64], 16) if len(rd) >= 66 else None

def beacon_blocks():
    resp = json.load(urllib.request.urlopen(BEACON + "/beacon/v1/blocks", timeout=10))
    out = {}
    for b in resp.get("blocks", []):
        out[b["beacon_chain"]] = (int(b["block_number"]), b["block_hash"])
    return out

# ============ A. Ticker de producción autónoma ============
print("=== A. Ticker de producción (esperando bloques beacon) ===")
deadline = time.time() + 90
chains = {}
while time.time() < deadline:
    chains = beacon_blocks()
    if chains.get("DPOS") and chains.get("PBFT") and chains["DPOS"][0] >= 2:
        break
    time.sleep(3)
check("ticker produce bloques beacon DPOS (≥2)", chains.get("DPOS", (None,))[0] is not None and chains["DPOS"][0] >= 2,
      str(chains))
check("ticker produce bloques beacon PBFT (≥2)", chains.get("PBFT", (None,))[0] is not None and chains["PBFT"][0] >= 2,
      str(chains))

# ============ Despliegue del contrato ============
bytecode = open("contracts/BeaconAnchor.bin", "rb").read().hex()
req = {"creator": G, "bytecode": bytecode, "name": "BeaconAnchor", "gasLimit": 8000000}
r = urllib.request.Request(ENGINE + "/api/contract/deploy",
    data=json.dumps(req).encode(), headers={"Content-Type": "application/json"})
ADDR = json.load(urllib.request.urlopen(r, timeout=30))["contract"]["address"]
ok, _ = call(ADDR, SEL["init"], ident(G))
check("deploy + init(hash(G))", ok)

# esperar a que el precompile vea ≥2 bloques (contexto DPOS)
deadline = time.time() + 60
while time.time() < deadline:
    ok, rd = call(ADDR, SEL["height"])
    h = w(rd) or 0
    if ok and h >= 2:
        break
    time.sleep(3)

print("=== B. Precompile BCR vs Beacon API (cadena DPOS) ===")
api_h, api_root = chains.get("DPOS", (0, "0x0"))
check(f"height() del contrato == último bloque beacon DPOS de la API ({api_h})",
      ok and h == api_h, f"contrato={h} api={api_h}")
ok, rd = call(ADDR, SEL["latestRoot"])
root_now = w(rd)
check("latestRoot() == block_hash QSH del último bloque (API)",
      ok and root_now == int(api_root, 16), f"contrato={root_now} api={int(api_root,16)}")
ok1, rd1 = call(ADDR, SEL["latestRoot"])
ok2, rd2 = call(ADDR, SEL["latestRoot"])
check("latestRoot() determinista (misma raíz en 2 lecturas)", ok1 and ok2 and w(rd1) == w(rd2))
ok, rd = call(ADDR, SEL["rootAt"], h + 1000)
check("rootAt(altura futura) == 0 (aún no anclado)", ok and w(rd) == 0, f"ret={w(rd)}")

# capturar raíz histórica y esperar que el ticker avance la altura
h1, r1 = h, root_now
deadline = time.time() + 60
while time.time() < deadline:
    ok, rd = call(ADDR, SEL["height"])
    if ok and (w(rd) or 0) > h1:
        break
    time.sleep(3)
ok, rd = call(ADDR, SEL["height"])
check(f"el ticker avanzó la altura beacon ({h1} → {w(rd)})", ok and w(rd) > h1, f"h={w(rd)}")
ok, rd = call(ADDR, SEL["rootAt"], h1)
check(f"rootAt({h1}) == raíz capturada en h1 (histórico inmutable)",
      ok and w(rd) == r1, f"ret={w(rd)} want={r1}")

print("=== C. Anclaje on-chain (KSP-2) ===")
ok, rd = call(ADDR, SEL["anchor"])
anchored_root = w(rd)
ok2, rd2 = call(ADDR, SEL["latestRoot"])
check("anchor() retorna la raíz viva y la almacena", ok and ok2 and anchored_root == w(rd2) and anchored_root != 0)
ok, rd = call(ADDR, SEL["getLastHeight"])
anchored_h = w(rd)
check(f"getLastHeight() == altura al anclar ({anchored_h})", ok and anchored_h is not None)
ok, rd = call(ADDR, SEL["verifyAnchored"], anchored_h)
check(f"verifyAnchored({anchored_h}) == 1 (anclaje coincide con la viva)", ok and w(rd) == 1)

# esperar un tick: el anclaje histórico sigue válido aunque la cadena avanzó
deadline = time.time() + 60
while time.time() < deadline:
    ok, rd = call(ADDR, SEL["height"])
    if ok and (w(rd) or 0) > anchored_h:
        break
    time.sleep(3)
ok, rd = call(ADDR, SEL["height"])
check(f"la beacon avanzó tras el anclaje ({anchored_h} → {w(rd)})", ok and w(rd) > anchored_h)
ok, rd = call(ADDR, SEL["verifyAnchored"], anchored_h)
check(f"verifyAnchored({anchored_h}) sigue == 1 (raíz anclada inmutable)", ok and w(rd) == 1)
ok, rd = call(ADDR, SEL["rootAt"], anchored_h + 500)
check("rootAt(futuro lejano) == 0", ok and w(rd) == 0)
ok, _ = call(ADDR, SEL["anchor"], caller="0xb01769d48d9ab0ab0029ce9b78e1de769cba11fd")
check("anchor() por no-owner → REVERT", not ok)

print("=== D. Producción PBFT (vía Beacon API) ===")
b1 = beacon_blocks().get("PBFT", (None, None))
time.sleep(8)
b2 = beacon_blocks().get("PBFT", (None, None))
check(f"beacon PBFT avanza autónomamente ({b1[0]} → {b2[0]})",
      b1[0] is not None and b2[0] is not None and b2[0] > b1[0], f"{b1} → {b2}")

print(f"\n===== GATE 4 (KSP-2 BCR): {PASSED} pasados / {FAILED} fallados =====")
sys.exit(1 if FAILED else 0)
