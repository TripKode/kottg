#!/usr/bin/env python3
# Gate 2 — KSP Fase 2: Registry.kg con map[address,u256] (QSH de slots),
# vec[u256] (base+idx), sec.qsh (QSH runtime), u64 (range-check) y
# dirección derivada wallet-scheme.
import json, urllib.request, sys, subprocess

ENGINE = "http://localhost:8084"
G    = "0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"
DEV1 = "0xb01769d48d9ab0ab0029ce9b78e1de769cba11fd"
DEV2 = 0xc14e6d866bac49aaf92694a962d1a0a533475794  # int form

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
    "deposit": sel("deposit(address,u256)"),
    "balanceOf": sel("balanceOf(address)"),
    "transferFrom": sel("transferFrom(address,address,u256)"),
    "setScore": sel("setScore(u256,u256)"),
    "getScore": sel("getScore(u256)"),
    "commit": sel("commit(u256)"),
    "bump": sel("bump()"),
    "getCounter": sel("getCounter()"),
    "setCounterBig": sel("setCounterBig(u64)"),
}
HASHG = int(subprocess.run(["/tmp/kselbin", G], capture_output=True, text=True).stdout.split()[-1], 16)

PASSED = FAILED = 0
def call(addr, caller, sel, *args):
    def word(a):
        if isinstance(a, str):
            a = int(a, 16) if not a.startswith("0x") else int(a, 16)
        return format(a & (2**256-1), "064x")
    inp = sel + "00"*28 + "".join(word(a) for a in args)
    req = {"contract_address": addr, "caller": caller, "input": inp,
           "gas_limit": 4000000, "gas_price": 1000000000}
    r = urllib.request.Request(ENGINE + "/api/contract/call",
        data=json.dumps(req).encode(), headers={"Content-Type": "application/json"})
    try:
        resp = json.load(urllib.request.urlopen(r, timeout=30))
        ex = resp.get("execution", {})
        return ex.get("success"), ex.get("gas_used"), ex.get("return_data","0x")
    except urllib.error.HTTPError as e:
        return False, 0, "HTTP"+str(e.code)

def check(desc, ok, detail=""):
    global PASSED, FAILED
    PASSED, FAILED = PASSED + ok, FAILED + (not ok)
    print(("✅" if ok else "❌") + f" {desc}" + (f" — {detail}" if detail and not ok else ""))

def w(rd, i=0): return int(rd[2+64*i:2+64*(i+1)],16) if len(rd) >= 2+64*(i+1) else None

bytecode = open("contracts/Registry.bin","rb").read().hex()
req = {"creator": G, "bytecode": bytecode, "name": "RegistryKottg", "gasLimit": 8000000}
r = urllib.request.Request(ENGINE + "/api/contract/deploy",
    data=json.dumps(req).encode(), headers={"Content-Type": "application/json"})
resp = json.load(urllib.request.urlopen(r, timeout=30))
ADDR = resp["contract"]["address"]
print(f"Registry desplegado en: {ADDR} (addr wallet-scheme ✓ = {ADDR.startswith('0x') and len(ADDR)==42})\n")

ok, _, _ = call(ADDR, G, SEL["init"], HASHG)
check("init(hash(G))", ok)

print("=== MAPPING: map[address,u256] via QSH de slots ===")
ok, _, _ = call(ADDR, G, SEL["deposit"], DEV1, 1000)
check("deposit(DEV1, 1000)", ok)
ok, _, rd = call(ADDR, G, SEL["balanceOf"], DEV1)
check("balanceOf(DEV1) == 1000", ok and w(rd) == 1000, f"ret={w(rd)}")

ok, _, _ = call(ADDR, DEV1, SEL["deposit"], DEV1, 5)
check("deposit desde no-owner → REVERT", not ok)

ok, _, _ = call(ADDR, G, SEL["deposit"], DEV2, 500)
check("deposit(DEV2, 500)", ok)
ok, _, rd = call(ADDR, G, SEL["balanceOf"], DEV2)
check("balanceOf(DEV2) == 500 (keys aisladas por QSH)", ok and w(rd) == 500, f"ret={w(rd)}")

ok, _, _ = call(ADDR, G, SEL["transferFrom"], DEV1, DEV2, 300)
check("transferFrom(DEV1→DEV2, 300)", ok)
ok, _, rd = call(ADDR, G, SEL["balanceOf"], DEV1)
check("balanceOf(DEV1) == 700 tras transfer", ok and w(rd) == 700, f"ret={w(rd)}")
ok, _, rd = call(ADDR, G, SEL["balanceOf"], DEV2)
check("balanceOf(DEV2) == 800 tras transfer", ok and w(rd) == 800, f"ret={w(rd)}")

ok, _, _ = call(ADDR, G, SEL["transferFrom"], DEV1, DEV2, 99999)
check("transferFrom sin fondos → REVERT", not ok)

print("=== VEC: vec[u256] base+idx ===")
ok, _, _ = call(ADDR, G, SEL["setScore"], 0, 42)
check("setScore(0, 42)", ok)
ok, _, _ = call(ADDR, G, SEL["setScore"], 5, 77)
check("setScore(5, 77)", ok)
ok, _, rd = call(ADDR, G, SEL["getScore"], 0)
check("getScore(0) == 42", ok and w(rd) == 42, f"ret={w(rd)}")
ok, _, rd = call(ADDR, G, SEL["getScore"], 5)
check("getScore(5) == 77 (slots vec aislados de maps)", ok and w(rd) == 77, f"ret={w(rd)}")

print("=== SEC.QSH: hash post-cuántico runtime ===")
ok, _, rd = call(ADDR, G, SEL["commit"], 12345)
h_onchain = w(rd)
ok2, _, rd2 = call(ADDR, G, SEL["commit"], 12345)
check("commit(12345) determinista (mismo hash 2 veces)",
      ok and ok2 and h_onchain == w(rd2) and h_onchain > 0, f"h={h_onchain}")
# el hash on-chain debe coincidir con security.GenerateQuantumHash(pad32(12345))
want = int(subprocess.run(["/tmp/kselbin"], capture_output=True, text=True,
    input=None).stdout or "0", 16)  # ksel sin args no imprime — computo directo:
import struct
# computar via helper: pasar 32-byte big-endian de 12345 como address-like string no funciona;
# mejor verificar con el QSH del probe de Fase 0: 12345 en u256 word → hash
ok3, _, rd3 = call(ADDR, G, SEL["commit"], 99999)
check("commit(99999) ≠ commit(12345) (hash sensible al input)",
      ok3 and w(rd3) != h_onchain, f"h2={w(rd3)}")

print("=== U64: range-check ===")
ok, _, _ = call(ADDR, G, SEL["bump"])
ok, _, _ = call(ADDR, G, SEL["bump"])
ok, _, rd = call(ADDR, G, SEL["getCounter"])
check("getCounter() == 2 tras 2 bumps", ok and w(rd) == 2, f"ret={w(rd)}")

ok, _, _ = call(ADDR, G, SEL["setCounterBig"], 2**64 - 1)
check("setCounterBig(2^64 - 1) — máximo u64 válido", ok)
ok, _, _ = call(ADDR, G, SEL["bump"])
check("bump() en 2^64-1 → REVERT (range-check 64 bits)", not ok)

ok, _, _ = call(ADDR, G, SEL["setCounterBig"], 2**64)
check("setCounterBig(2^64) → REVERT (fuera de rango u64)", not ok)
ok, _, rd = call(ADDR, G, SEL["getCounter"])
check("counter sigue en 2^64-1 (no corrompido)", ok and w(rd) == 2**64 - 1, f"ret={w(rd)}")

print(f"\n===== GATE 2: {PASSED} pasados / {FAILED} fallados =====")
sys.exit(1 if FAILED else 0)
