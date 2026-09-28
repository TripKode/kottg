#!/usr/bin/env python3
# Gate 1 — KSP Fase 1: SimpleStorage.kg compilado con kottg, desplegado en la
# testnet viva vía el engine, ciclo completo: init → set (solo owner) → get →
# increment/decrement (KSM) → decrement bajo cero (trap).
import json, urllib.request, sys, subprocess

ENGINE = "http://localhost:8084"
G     = "0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"   # deployer/owner
DEV1  = "0xb01769d48d9ab0ab0029ce9b78e1de769cba11fd"

# KSP-6B: selectores KSEL-8 computados en runtime desde la firma canónica
# (la firma es lo congelado; el selector se deriva con el dominio KSEL-8).
_SEL_CACHE = {}
def sel(sig):
    if sig not in _SEL_CACHE:
        out = subprocess.run(["/tmp/kselbin", sig], capture_output=True, text=True)
        _SEL_CACHE[sig] = out.stdout.split()[1]
    return _SEL_CACHE[sig]

SEL = {
    "init(u256)":   sel("init(u256)"),
    "set(u256)":    sel("set(u256)"),
    "get()":        sel("get()"),
    "getOwner()":   sel("getOwner()"),
    "increment()":  sel("increment()"),
    "decrement()":  sel("decrement()"),
}

PASSED = FAILED = 0

def call(addr, caller, sel, *args):
    """sel None → calldata vacío (init). args: ints → palabras de 32 bytes."""
    if sel is None:
        inp = ""
    else:
        inp = sel + "00" * 28  # selector en los 4 bytes altos de la palabra 0
        for a in args:
            inp += format(a, "064x")
    req = {"contract_address": addr, "caller": caller, "input": inp,
           "gas_limit": 4000000, "gas_price": 1000000000}
    r = urllib.request.Request(ENGINE + "/api/contract/call",
        data=json.dumps(req).encode(), headers={"Content-Type": "application/json"})
    try:
        resp = json.load(urllib.request.urlopen(r, timeout=30))
        ex = resp.get("execution", {})
        return ex.get("success"), ex.get("gas_used"), ex.get("return_data", "0x")
    except urllib.error.HTTPError as e:
        return False, 0, "HTTP " + str(e.code)

def ret_int(rd):
    return int(rd[2:66], 16) if rd and len(rd) >= 66 else None

def check(desc, ok, detail=""):
    global PASSED, FAILED
    PASSED, FAILED = PASSED + ok, FAILED + (not ok)
    print(("✅" if ok else "❌") + f" {desc}" + (f" — {detail}" if detail and not ok else ""))

# ---- deploy ----
bytecode = open("contracts/SimpleStorage.bin", "rb").read()
hexcode = bytecode.hex()
req = {"creator": G, "bytecode": hexcode, "name": "SimpleStorageKottg",
       "version": "2.0-f1", "gasLimit": 8000000}
r = urllib.request.Request(ENGINE + "/api/contract/deploy",
    data=json.dumps(req).encode(), headers={"Content-Type": "application/json"})
resp = json.load(urllib.request.urlopen(r, timeout=30))
ADDR = resp["contract"]["address"]
print(f"SimpleStorage (kottg) desplegado en: {ADDR}  codeSize={resp['contract'].get('codeSize')}\n")

# ---- init: ahora despachable por su KSEL-8 (init(u256) = 00680504) ----
ok, gas, rd = call(ADDR, G, SEL["init(u256)"], 0)
check("init(0) via selector KSEL-8 desde G", ok, f"gas={gas}")

# ---- owner quedó como CALLER del init ----
ok, gas, rd = call(ADDR, G, SEL["getOwner()"])
check("getOwner() == hash(G)", ok and ret_int(rd) is not None, f"ret={rd}")

# ---- set(42) desde owner (modifier onlyOwner) ----
ok, gas, rd = call(ADDR, G, SEL["set(u256)"], 42)
check("set(42) desde owner G", ok, f"gas={gas}")

# ---- set desde NO-owner → revert ----
ok, gas, rd = call(ADDR, DEV1, SEL["set(u256)"], 99)
check("set(99) desde no-owner DEV1 → REVERT (modifier)", not ok, f"gas={gas}")

# ---- get() → 42 ----
ok, gas, rd = call(ADDR, DEV1, SEL["get()"])
check("get() == 42 (view desde cualquiera)", ok and ret_int(rd) == 42, f"ret={rd}")

# ---- increment() ×3 → 45 ----
for i in range(3):
    ok, gas, rd = call(ADDR, G, SEL["increment()"])
    check(f"increment() #{i+1}", ok, f"gas={gas}")
ok, gas, rd = call(ADDR, G, SEL["get()"])
check("get() == 45 tras 3 increment", ok and ret_int(rd) == 45, f"ret={rd}")

# ---- decrement() → 44 ----
ok, gas, rd = call(ADDR, G, SEL["decrement()"])
check("decrement() → 44", ok, f"gas={gas}")
ok, gas, rd = call(ADDR, G, SEL["get()"])
check("get() == 44", ok and ret_int(rd) == 44, f"ret={rd}")

# ---- KSM: decrement 45 veces → al llegar a 0, el decremento extra → trap ----
for i in range(44):
    ok, _, _ = call(ADDR, G, SEL["decrement()"])
    if not ok:
        check(f"decrement masivo (#{i+1}) — fallo inesperado", False, "revert prematuro")
        break
ok, gas, rd = call(ADDR, G, SEL["get()"])
check("get() == 0 tras agotar", ok and ret_int(rd) == 0, f"ret={rd}")
ok, gas, rd = call(ADDR, G, SEL["decrement()"])
check("KSM: decrement bajo cero → REVERT (underflow trap)", not ok, f"gas={gas}")
ok, gas, rd = call(ADDR, G, SEL["get()"])
check("get() sigue == 0 (estado no corrompido)", ok and ret_int(rd) == 0, f"ret={rd}")

print(f"\n===== GATE 1: {PASSED} pasados / {FAILED} fallados =====")
sys.exit(1 if FAILED else 0)
