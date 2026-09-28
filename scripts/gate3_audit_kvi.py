#!/usr/bin/env python3
# KSP-1 Gate 3 — Auditoría KVI (Kodechain Value Instrument)
# 44 casos: despliegue, init-guard, KVI-M mint, transfer, approve/transferFrom
# (allowances anidadas map[address,map[address,u256]]), atomicidad de reverts,
# conservación de oferta, aislamiento de claves y KSM overflow.
#
# Identidad KVI = QSH completo de la dirección (lo que emite CALLER).
import json, urllib.request, sys, subprocess

ENGINE  = "http://localhost:8084"
FAUCET  = "0x66d1acb3f9065eabecd97b0c2df07de45245518f"  # minter KVI-M (G)
ALICE_A = "0xb01769d48d9ab0ab0029ce9b78e1de769cba11fd"
BOB_A   = "0xc14e6d866bac49aaf92694a962d1a0a533475794"
CAROL_A = "0xb6af90229c4fc6bc44b679adbee4b8e17dd32ec3"
DAVE_A  = "0x199ece00b5a4b7d9ba4df2b6a2f3d84bf5d18a2c"
EVE_A   = "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"  # nunca tocada

# KSEL-8 congelados KSP-1 (docs/Operations/KSP-1-KVI.md)
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
    "mint": sel("mint(address,u256)"),
    "transfer": sel("transfer(address,u256)"),
    "approve": sel("approve(address,u256)"),
    "transferFrom": sel("transferFrom(address,address,u256)"),
    "totalSupply": sel("totalSupply()"),
    "balanceOf": sel("balanceOf(address)"),
    "allowance": sel("allowance(address,address)"),
}

def ident(addr):
    out = subprocess.run(["/tmp/kselbin", addr], capture_output=True, text=True)
    return int(out.stdout.split()[-1], 16)

PASSED = FAILED = 0
def check(desc, ok, detail=""):
    global PASSED, FAILED
    PASSED, FAILED = PASSED + bool(ok), FAILED + (not ok)
    print(("✅" if ok else "❌") + " " + desc + ("" if ok or not detail else f" — {detail}"))

def call(addr, caller, sel, *args):
    def word(a):
        if isinstance(a, str): a = int(a, 16)
        return format(a & (2**256 - 1), "064x")
    inp = sel + "00" * 28 + "".join(word(a) for a in args)
    req = {"contract_address": addr, "caller": caller, "input": inp,
           "gas_limit": 4000000, "gas_price": 1000000000}
    r = urllib.request.Request(ENGINE + "/api/contract/call",
        data=json.dumps(req).encode(), headers={"Content-Type": "application/json"})
    try:
        ex = json.load(urllib.request.urlopen(r, timeout=30)).get("execution", {})
        return ex.get("success"), ex.get("return_data", "0x")
    except urllib.error.HTTPError as e:
        return False, "HTTP" + str(e.code)

def w(rd, i=0):
    return int(rd[2 + 64*i: 2 + 64*(i+1)], 16) if len(rd) >= 2 + 64*(i+1) else None

FAUCET_I, ALICE_I, BOB_I = ident(FAUCET), ident(ALICE_A), ident(BOB_A)
CAROL_I, DAVE_I, EVE_I  = ident(CAROL_A), ident(DAVE_A), ident(EVE_A)
M = 2**256 - 1

# ============ A. Despliegue e init ============
print("=== A. Despliegue e init ===")
bytecode = open("contracts/ValueInstrument.bin", "rb").read().hex()
req = {"creator": FAUCET, "bytecode": bytecode, "name": "KVI", "gasLimit": 8000000}
r = urllib.request.Request(ENGINE + "/api/contract/deploy",
    data=json.dumps(req).encode(), headers={"Content-Type": "application/json"})
resp = json.load(urllib.request.urlopen(r, timeout=30))
ADDR = resp["contract"]["address"]
check("despliegue OK — dirección wallet-scheme (42 chars)",
      ADDR.startswith("0x") and len(ADDR) == 42, ADDR)

ok, _ = call(ADDR, FAUCET, SEL["init"], FAUCET_I)
check("init(minter=FAUCET) OK", ok)
ok, _ = call(ADDR, FAUCET, SEL["init"], FAUCET_I)
check("re-init → REVERT (guarda initialized)", not ok)
ok, rd = call(ADDR, FAUCET, SEL["totalSupply"])
check("totalSupply() == 0 tras init", ok and w(rd) == 0, f"ret={w(rd)}")
ok, rd = call(ADDR, FAUCET, SEL["balanceOf"], EVE_I)
check("balanceOf(identidad virgen) == 0", ok and w(rd) == 0, f"ret={w(rd)}")

# ============ B. KVI-M: mint ============
print("=== B. KVI-M: acuñación ===")
ok, _ = call(ADDR, ALICE_A, SEL["mint"], ALICE_I, 1000)
check("mint por NO-minter → REVERT", not ok)
ok, rd = call(ADDR, FAUCET, SEL["mint"], ALICE_I, 1000)
check("mint(ALICE, 1000) OK, retorna 1 (KVI_OK)", ok and w(rd) == 1)
ok, rd = call(ADDR, FAUCET, SEL["balanceOf"], ALICE_I)
check("balanceOf(ALICE) == 1000", ok and w(rd) == 1000, f"ret={w(rd)}")
ok, rd = call(ADDR, FAUCET, SEL["totalSupply"])
check("totalSupply() == 1000", ok and w(rd) == 1000, f"ret={w(rd)}")
ok, _ = call(ADDR, FAUCET, SEL["mint"], BOB_I, 500)
check("mint(BOB, 500) OK", ok)
ok, rd = call(ADDR, FAUCET, SEL["totalSupply"])
check("totalSupply() == 1500", ok and w(rd) == 1500, f"ret={w(rd)}")
ok, _ = call(ADDR, FAUCET, SEL["mint"], 0, 1)
check("mint(identidad cero, 1) → REVERT", not ok)
ok, _ = call(ADDR, FAUCET, SEL["mint"], ALICE_I, M - 1000)
check("mint que desborda saldo (KSM) → REVERT", not ok)
ok, rd = call(ADDR, FAUCET, SEL["balanceOf"], ALICE_I)
check("balanceOf(ALICE) intacto (1000) tras intento de overflow", ok and w(rd) == 1000, f"ret={w(rd)}")

# ============ C. transfer ============
print("=== C. transfer ===")
ok, _ = call(ADDR, CAROL_A, SEL["transfer"], BOB_I, 1)
check("transfer sin saldo → REVERT", not ok)
ok, _ = call(ADDR, ALICE_A, SEL["transfer"], ALICE_I, 100)
check("self-transfer(ALICE→ALICE, 100) OK", ok)
ok, rd = call(ADDR, ALICE_A, SEL["balanceOf"], ALICE_I)
check("saldo sin cambio tras self-transfer (1000)", ok and w(rd) == 1000, f"ret={w(rd)}")
ok, rd = call(ADDR, ALICE_A, SEL["transfer"], BOB_I, 400)
check("transfer(ALICE→BOB, 400) OK, retorna 1", ok and w(rd) == 1)
ok, rd = call(ADDR, ALICE_A, SEL["balanceOf"], ALICE_I)
check("balanceOf(ALICE) == 600", ok and w(rd) == 600, f"ret={w(rd)}")
ok, rd = call(ADDR, ALICE_A, SEL["balanceOf"], BOB_I)
check("balanceOf(BOB) == 900", ok and w(rd) == 900, f"ret={w(rd)}")
ok, rd = call(ADDR, FAUCET, SEL["totalSupply"])
check("totalSupply() == 1500 tras transfer (conservación)", ok and w(rd) == 1500, f"ret={w(rd)}")
ok, _ = call(ADDR, ALICE_A, SEL["transfer"], BOB_I, 601)
check("transfer por encima del saldo → REVERT", not ok)
ok, rd = call(ADDR, ALICE_A, SEL["balanceOf"], ALICE_I)
check("saldo ALICE intacto tras fallo (atomicidad)", ok and w(rd) == 600, f"ret={w(rd)}")
ok, _ = call(ADDR, ALICE_A, SEL["transfer"], 0, 10)
check("transfer a identidad cero → REVERT", not ok)
ok, _ = call(ADDR, ALICE_A, SEL["transfer"], BOB_I, 0)
check("transfer de 0 OK (no-op permitido)", ok)
ok, rd = call(ADDR, ALICE_A, SEL["balanceOf"], ALICE_I)
check("saldo sin cambio tras transfer 0", ok and w(rd) == 600, f"ret={w(rd)}")

# ============ D. approve / transferFrom ============
print("=== D. approve / transferFrom (allowances anidadas) ===")
ok, _ = call(ADDR, ALICE_A, SEL["approve"], 0, 1)
check("approve a identidad cero → REVERT", not ok)
ok, _ = call(ADDR, ALICE_A, SEL["approve"], BOB_I, 300)
check("ALICE approve(BOB, 300) OK", ok)
ok, rd = call(ADDR, ALICE_A, SEL["allowance"], ALICE_I, BOB_I)
check("allowance(ALICE→BOB) == 300", ok and w(rd) == 300, f"ret={w(rd)}")
ok, _ = call(ADDR, BOB_A, SEL["transferFrom"], ALICE_I, CAROL_I, 100)
check("BOB transferFrom(ALICE→CAROL, 100) OK", ok)
ok, rd = call(ADDR, ALICE_A, SEL["balanceOf"], ALICE_I)
check("balanceOf(ALICE) == 500 tras transferFrom", ok and w(rd) == 500, f"ret={w(rd)}")
ok, rd = call(ADDR, CAROL_A, SEL["balanceOf"], CAROL_I)
check("balanceOf(CAROL) == 100", ok and w(rd) == 100, f"ret={w(rd)}")
ok, rd = call(ADDR, ALICE_A, SEL["allowance"], ALICE_I, BOB_I)
check("allowance(ALICE→BOB) == 200 tras gasto", ok and w(rd) == 200, f"ret={w(rd)}")
ok, _ = call(ADDR, BOB_A, SEL["transferFrom"], ALICE_I, CAROL_I, 201)
check("transferFrom excediendo allowance → REVERT", not ok)
ok, rd = call(ADDR, ALICE_A, SEL["balanceOf"], ALICE_I)
check("saldo ALICE intacto tras allowance-fail (atomicidad)", ok and w(rd) == 500, f"ret={w(rd)}")
ok, _ = call(ADDR, CAROL_A, SEL["transferFrom"], ALICE_I, CAROL_I, 1)
check("transferFrom SIN allowance → REVERT", not ok)
ok, _ = call(ADDR, BOB_A, SEL["transferFrom"], ALICE_I, BOB_I, 200)
check("transferFrom consume el resto de allowance (200) OK", ok)
ok, rd = call(ADDR, ALICE_A, SEL["allowance"], ALICE_I, BOB_I)
check("allowance(ALICE→BOB) == 0 tras consumir", ok and w(rd) == 0, f"ret={w(rd)}")
ok, _ = call(ADDR, BOB_A, SEL["transferFrom"], ALICE_I, CAROL_I, 1)
check("transferFrom con allowance agotada → REVERT", not ok)

# balance insuficiente CON allowance vigente
# (ALICE tiene 300 aquí: 1000 − 400 transfer − 100 transferFrom − 200 transferFrom)
ok, _ = call(ADDR, ALICE_A, SEL["approve"], BOB_I, 50)
check("ALICE approve(BOB, 50) OK", ok)
ok, _ = call(ADDR, ALICE_A, SEL["transfer"], DAVE_I, 300)
check("ALICE vacía su saldo (transfer 300 → DAVE)", ok)
ok, rd = call(ADDR, ALICE_A, SEL["balanceOf"], ALICE_I)
check("balanceOf(ALICE) == 0 tras vaciar", ok and w(rd) == 0, f"ret={w(rd)}")
ok, _ = call(ADDR, BOB_A, SEL["transferFrom"], ALICE_I, BOB_I, 50)
check("transferFrom con allowance OK pero saldo 0 → REVERT", not ok)
ok, rd = call(ADDR, DAVE_A, SEL["balanceOf"], DAVE_I)
check("balanceOf(DAVE) == 300 recibidos", ok and w(rd) == 300, f"ret={w(rd)}")

# ============ E. Aislamiento de claves y determinismo ============
print("=== E. Aislamiento y determinismo ===")
ok, rd = call(ADDR, BOB_A, SEL["allowance"], BOB_I, ALICE_I)
check("allowance direccional: (BOB→ALICE) == 0 ≠ (ALICE→BOB)=50", ok and w(rd) == 0, f"ret={w(rd)}")
ok, rd = call(ADDR, CAROL_A, SEL["allowance"], CAROL_I, BOB_I)
check("allowance claves anidadas aisladas (CAROL→BOB) == 0", ok and w(rd) == 0, f"ret={w(rd)}")
r1 = call(ADDR, ALICE_A, SEL["balanceOf"], ALICE_I)
r2 = call(ADDR, ALICE_A, SEL["balanceOf"], ALICE_I)
check("balanceOf determinista (2 lecturas iguales)", r1 == r2 and r1[0])
ok, rd = call(ADDR, EVE_A, SEL["balanceOf"], EVE_I)
check("identidad virgen EVE == 0 (claves QSH no colisionan)", ok and w(rd) == 0, f"ret={w(rd)}")

# ============ F. Conservación global ============
print("=== F. Conservación final ===")
total = 0
for who, idt in [("ALICE", ALICE_I), ("BOB", BOB_I), ("CAROL", CAROL_I),
                 ("DAVE", DAVE_I), ("EVE", EVE_I), ("FAUCET", FAUCET_I)]:
    ok, rd = call(ADDR, FAUCET, SEL["balanceOf"], idt)
    total += w(rd) or 0
ok, rd = call(ADDR, FAUCET, SEL["totalSupply"])
sup = w(rd)
check(f"Σ balances ({total}) == totalSupply ({sup})", ok and total == sup,
      f"Σ={total} supply={sup}")
check("totalSupply nunca cambió por transfers (1500)", sup == 1500, f"sup={sup}")

print(f"\n===== GATE 3 (KSP-1 KVI): {PASSED} pasados / {FAILED} fallados =====")
sys.exit(1 if FAILED else 0)
