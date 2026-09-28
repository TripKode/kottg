#!/usr/bin/env python3
# KSP-3 Gate 5 — AMM KLP/CPV: factory + CREATE on-chain, CALL entre
# contratos con persistencia real, atomicidad cross-contrato (journal
# compartido), producto constante con fee, slippage y burn-by-custody.
# Toda la matemática del pool se espeja EXACTAMENTE en Python (DIV floor).
import json, urllib.request, subprocess, sys

ENGINE = "http://localhost:8084"
G      = "0x66d1acb3f9065eabecd97b0c2df07de45245518f"
ALICE  = "0xb01769d48d9ab0ab0029ce9b78e1de769cba11fd"
BOB    = "0xc14e6d866bac49aaf92694a962d1a0a533475794"
CAROL  = "0xb6af90229c4fc6bc44b679adbee4b8e17dd32ec3"
MAXU   = 2**256 - 1

def ident(a):
    return int(subprocess.run(["/tmp/kselbin", a], capture_output=True, text=True).stdout.split()[-1], 16)

def w40(n): return "0x%040x" % n

# KSP-6B: selectores KSEL-8 computados en runtime desde la firma canónica
# (la firma es lo congelado; el selector se deriva con el dominio KSEL-8).
_SEL_CACHE = {}
def sel(sig):
    if sig not in _SEL_CACHE:
        out = subprocess.run(["/tmp/kselbin", sig], capture_output=True, text=True)
        _SEL_CACHE[sig] = out.stdout.split()[1]
    return _SEL_CACHE[sig]

KVI = {"init": sel("init(address)"), "mint": sel("mint(address,u256)"),
       "transfer": sel("transfer(address,u256)"), "approve": sel("approve(address,u256)"),
       "transferFrom": sel("transferFrom(address,address,u256)"),
       "totalSupply": sel("totalSupply()"), "balanceOf": sel("balanceOf(address)"),
       "allowance": sel("allowance(address,address)")}
FAC = {"init": sel("init(address)"), "createPool": sel("createPool(address,address)"),
       "poolAt": sel("poolAt(u256)"), "numPools": sel("numPools()")}
CPV = {"init": sel("init(address,address,address)"), "addLiquidity": sel("addLiquidity(u256,u256)"),
       "removeLiquidity": sel("removeLiquidity(u256)"),
       "swapAForB": sel("swapAForB(u256,u256)"), "swapBForA": sel("swapBForA(u256,u256)"),
       "aReserve": sel("aReserve()"), "bReserve": sel("bReserve()"),
       "aToken": sel("aToken()"), "bToken": sel("bToken()"),
       "getKlp": sel("getKlp()"), "getLpTotal": sel("getLpTotal()"),
       "quoteAForB": sel("quoteAForB(u256)"), "quoteBForA": sel("quoteBForA(u256)")}

PASSED = FAILED = 0
def check(desc, ok, detail=""):
    global PASSED, FAILED
    PASSED, FAILED = PASSED + bool(ok), FAILED + (not ok)
    print(("✅" if ok else "❌") + " " + desc + ("" if ok or not detail else f" — {detail}"))

def call(addr, sel, *args, caller=G, gas=4000000):
    def word(a):
        return format((int(a, 16) if isinstance(a, str) else a) & MAXU, "064x")
    inp = sel + "00"*28 + "".join(word(a) for a in args)
    req = {"contract_address": addr, "caller": caller, "input": inp,
           "gas_limit": gas, "gas_price": 1000000000}
    r = urllib.request.Request(ENGINE + "/api/contract/call",
        data=json.dumps(req).encode(), headers={"Content-Type": "application/json"})
    try:
        ex = json.load(urllib.request.urlopen(r, timeout=60)).get("execution", {})
        return ex.get("success"), ex.get("return_data", "0x")
    except urllib.error.HTTPError as e:
        return False, "HTTP" + str(e.code)

def w(rd):
    return int(rd[2:2+64], 16) if rd and len(rd) >= 66 else None

def deploy(name):
    bc = open(f"contracts/{name}.bin", "rb").read().hex()
    req = {"creator": G, "bytecode": bc, "name": name, "gasLimit": 8000000}
    r = urllib.request.Request(ENGINE + "/api/contract/deploy",
        data=json.dumps(req).encode(), headers={"Content-Type": "application/json"})
    return json.load(urllib.request.urlopen(r, timeout=30))["contract"]["address"]

# ================= DESPLIEGUE =================
KVIA  = deploy("ValueInstrument")   # token A
KVIB  = deploy("ValueInstrument")   # token B
FACAD = deploy("KLPFactory")
IA, IB, IG = ident(ALICE), ident(BOB), ident(G)
for a in (KVIA, KVIB):
    call(a, KVI["init"], IG)
call(KVIA, KVI["mint"], IA, 10000); call(KVIA, KVI["mint"], IB, 5000)
call(KVIB, KVI["mint"], IA, 40000); call(KVIB, KVI["mint"], IB, 500)
call(FACAD, FAC["init"], IG)
print(f"KVIA: {KVIA}\nKVIB: {KVIB}\nFactory: {FACAD}\n")

# ============ A. CREATE y registro ============
print("=== A. CREATE (opcode) + registro on-chain ===")
ok, rd = call(FACAD, FAC["createPool"], int(KVIA, 16), int(KVIB, 16), gas=8000000)
POOL = w40(w(rd) or 0)
check("factory.createPool() via CREATE×2 + CALLs", ok and POOL != w40(0), f"pool={POOL}")
PI = ident(POOL)  # identidad KVI del pool
ok, rd = call(FACAD, FAC["numPools"])
check("numPools() == 1", ok and w(rd) == 1)
ok, rd = call(FACAD, FAC["poolAt"], 0)
check("poolAt(0) == pool creado", ok and w40(w(rd) or 0) == POOL)
ok, rd = call(POOL, CPV["aToken"])
check("pool registrado y llamable por la API: aToken() == KVIA",
      ok and w40(w(rd) or 0) == KVIA, f"ret={w(rd)}")
ok, rd = call(POOL, CPV["getKlp"])
KLP = w40(w(rd) or 0)
check("KLP creado via CREATE y wired al pool", ok and KLP != w40(0), f"klp={KLP}")
KI = ident(KLP)
ok, rd = call(POOL, CPV["aReserve"])
check("pool parte con reservas 0", ok and w(rd) == 0)
ok, _ = call(POOL, CPV["init"], int(KVIA,16), int(KVIB,16), int(KLP,16))
check("re-init del pool → REVERT (guarda initialized)", not ok)
ok, _ = call(KLP, KVI["mint"], IB, 5, caller=BOB)
check("mint del KLP por NO-minter (BOB) → REVERT (minter=pool)", not ok)

# ============ B. Liquidez ============
print("=== B. addLiquidity (CALL pool→KVI con persistencia real) ===")
call(KVIA, KVI["approve"], PI, MAXU, caller=ALICE)
call(KVIB, KVI["approve"], PI, MAXU, caller=ALICE)
ok, rd = call(POOL, CPV["addLiquidity"], 1000, 4000, caller=ALICE)
check("addLiquidity(1000, 4000) → 4000 KLP (A*B/1000)", ok and w(rd) == 4000, f"ret={w(rd)}")
ok, rd = call(POOL, CPV["aReserve"]); rA = w(rd)
ok2, rd = call(POOL, CPV["bReserve"]); rB = w(rd)
check("reservas (1000, 4000)", rA == 1000 and rB == 4000, f"({rA},{rB})")
ok, rd = call(KLP, KVI["balanceOf"], IA)
check("klp.balanceOf(ALICE) == 4000 — mint del POOL persistió via CALL", ok and w(rd) == 4000, f"ret={w(rd)}")
ok, rd = call(KLP, KVI["totalSupply"])
check("klp.totalSupply() == 4000", ok and w(rd) == 4000)
ok, rd = call(KVIA, KVI["balanceOf"], PI)
check("KVIA.balanceOf(ident(pool)) == 1000 (custodia)", ok and w(rd) == 1000, f"ret={w(rd)}")
ok, rd = call(KVIB, KVI["balanceOf"], PI)
check("KVIB.balanceOf(ident(pool)) == 4000 (custodia)", ok and w(rd) == 4000, f"ret={w(rd)}")

# ============ C. Atomicidad cross-contrato ============
print("=== C. ATOMICIDAD: revert del pool deshace el pull del KVI hijo ===")
# BOB solo aprueba KVIA (NO KVIB): el pull A COMMITTEA en el frame hijo,
# el pull B falla → el revert del pool debe DESHACER el pull A.
call(KVIA, KVI["approve"], PI, MAXU, caller=BOB)
ok, _ = call(POOL, CPV["addLiquidity"], 10, 10, caller=BOB)
check("addLiquidity sin approval de B → REVERT", not ok)
ok, rd = call(KVIA, KVI["balanceOf"], PI)
check("KVIA del pool INTACTO (1000) — pull A deshecho por el journal", ok and w(rd) == 1000, f"ret={w(rd)}")
ok, rd = call(KVIA, KVI["balanceOf"], IB)
check("saldo A de BOB intacto (5000)", ok and w(rd) == 5000, f"ret={w(rd)}")
ok, rd = call(POOL, CPV["aReserve"])
check("reservas del pool intactas tras el revert", ok and w(rd) == 1000)
ok, rd = call(KLP, KVI["totalSupply"])
check("klp.totalSupply() intacto (4000) — mint nunca ocurrió", ok and w(rd) == 4000)

# ============ D. Swaps (producto constante) ============
print("=== D. Swaps A↔B (x·y=k con fee 3/1000) ===")
call(KVIB, KVI["approve"], PI, MAXU, caller=BOB)

# espejo Python: out = rB*inFee // (rA*1000 + inFee); inFee = in*997
def out_a_for_b(rA, rB, amt): return rB * (amt * 997) // (rA * 1000 + amt * 997)
def out_b_for_a(rA, rB, amt): return rA * (amt * 997) // (rB * 1000 + amt * 997)

expect = out_a_for_b(1000, 4000, 100)
ok, rd = call(POOL, CPV["quoteAForB"], 100)
check(f"quoteAForB(100) == {expect} (fórmula espejada)", ok and w(rd) == expect, f"ret={w(rd)}")
ok, rd = call(POOL, CPV["swapAForB"], 100, 0, caller=BOB)
out1 = w(rd)
check(f"swapAForB(100, 0) == {expect}", ok and out1 == expect, f"ret={w(rd)}")
ok, rd = call(POOL, CPV["aReserve"]); nA = w(rd)
ok2, rd = call(POOL, CPV["bReserve"]); nB = w(rd)
check(f"reservas tras swap: ({1000+100}, {4000-out1})", nA == 1100 and nB == 4000 - expect,
      f"({nA},{nB})")
check(f"invariante k no decrece en swaps: {nA*nB} >= {1000*4000}", nA * nB >= 1000 * 4000)
ok, rd = call(KVIB, KVI["balanceOf"], IB)
check("BOB recibió exactamente out1 tokens B (500+362=862)", ok and w(rd) == 500 + expect, f"ret={w(rd)}")

ok, _ = call(POOL, CPV["swapAForB"], 100, expect + 1, caller=BOB)
check("slippage: minOut > out real → REVERT", not ok)
ok, _ = call(POOL, CPV["swapAForB"], 0, 0, caller=BOB)
check("swap de 0 → REVERT", not ok)
ok, _ = call(POOL, CPV["swapAForB"], 100, 0, caller=CAROL)
check("swap sin approval (CAROL) → REVERT y nada movido", not ok)
ok, rd = call(KVIA, KVI["balanceOf"], PI)
check("custodia A del pool intacta tras reverts (1100)", ok and w(rd) == 1100, f"ret={w(rd)}")

out2 = out_b_for_a(1100, 4000 - expect, 50)
ok, rd = call(POOL, CPV["swapBForA"], 50, 0, caller=BOB)
check(f"swapBForA(50, 0) == {out2} (dirección inversa)", ok and w(rd) == out2, f"ret={w(rd)}")
rA2, rB2 = 1100 - out2, 4000 - expect + 50
ok, rd = call(POOL, CPV["aReserve"]); gA = w(rd)
ok2, rd = call(POOL, CPV["bReserve"]); gB = w(rd)
check(f"reservas exactas tras swap inverso: ({rA2}, {rB2})", gA == rA2 and gB == rB2, f"({gA},{gB})")

# ============ E. removeLiquidity (burn-by-custody) ============
print("=== E. removeLiquidity ===")
call(KLP, KVI["approve"], PI, MAXU, caller=ALICE)
a_out = 1000 * rA2 // 4000
b_out = 1000 * rB2 // 4000
ok, rd = call(POOL, CPV["removeLiquidity"], 1000, caller=ALICE)
check(f"removeLiquidity(1000) → {a_out} A (proporcional)", ok and w(rd) == a_out, f"ret={w(rd)}")
ok, rd = call(KLP, KVI["balanceOf"], IA)
check("KLP de ALICE: 4000 → 3000", ok and w(rd) == 3000, f"ret={w(rd)}")
ok, rd = call(KLP, KVI["balanceOf"], PI)
check("KLP bajo custodia del pool == 1000 (burn-by-custody)", ok and w(rd) == 1000, f"ret={w(rd)}")
ok, rd = call(KLP, KVI["totalSupply"])
check("klp.totalSupply() == 4000 (custodia ≠ burn: KSP-1 congelado)", ok and w(rd) == 4000)
ok, rd = call(POOL, CPV["getLpTotal"])
check("lpTotal del pool == 3000", ok and w(rd) == 3000)
ok, rd = call(KVIA, KVI["balanceOf"], IA)
check(f"ALICE recibió {a_out} A (9271)", ok and w(rd) == 10000 - 1000 + a_out, f"ret={w(rd)}")
ok, rd = call(KVIB, KVI["balanceOf"], IA)
check(f"ALICE recibió {b_out} B (36922)", ok and w(rd) == 40000 - 4000 + b_out, f"ret={w(rd)}")
ok, _ = call(POOL, CPV["removeLiquidity"], 0, caller=ALICE)
check("removeLiquidity(0) → REVERT", not ok)
ok, _ = call(POOL, CPV["removeLiquidity"], 99999, caller=ALICE)
check("removeLiquidity > lpTotal → REVERT", not ok)

# ============ F. Registry de la factory ============
print("=== F. Factory: acceso yCREATE múltiple ===")
ok, _ = call(FACAD, FAC["createPool"], int(KVIA,16), int(KVIB,16), caller=ALICE)
check("createPool por no-gov → REVERT", not ok)
ok, rd = call(FACAD, FAC["createPool"], int(KVIB,16), int(KVIA,16), gas=8000000)
POOL2 = w40(w(rd) or 0)
check("segundo pool creado (nonce distinto → dirección distinta)",
      ok and POOL2 != w40(0) and POOL2 != POOL, f"p2={POOL2}")
ok, rd = call(FACAD, FAC["numPools"])
check("numPools() == 2", ok and w(rd) == 2)
ok, rd = call(FACAD, FAC["poolAt"], 1)
check("poolAt(1) == segundo pool", ok and w40(w(rd) or 0) == POOL2)
ok, rd = call(POOL2, CPV["aReserve"])
check("pool 2 independiente: reservas 0", ok and w(rd) == 0)

# ============ G. Conservación global ============
print("=== G. Conservación ===")
sumA = sumB = 0
for who in (IA, IB, PI):
    _, rda = call(KVIA, KVI["balanceOf"], who); sumA += w(rda) or 0
    _, rdb = call(KVIB, KVI["balanceOf"], who); sumB += w(rdb) or 0
check(f"Σ saldos A ({sumA}) == total acuñado (15000)", sumA == 15000, f"Σ={sumA}")
check(f"Σ saldos B ({sumB}) == total acuñado (40500)", sumB == 40500, f"Σ={sumB}")
ok, rd = call(POOL, CPV["aReserve"]); fA = w(rd)
ok2, rd = call(POOL, CPV["bReserve"]); fB = w(rd)
check(f"reservas == custodia real del pool ({fA}, {fB})",
      ok and ok2 and fA == 1000 + 100 - out2 - a_out and fB == 4000 - out1 + 50 - b_out,
      f"({fA},{fB})")

print(f"\n===== GATE 5 (KSP-3 KLP/CPV): {PASSED} pasados / {FAILED} fallados =====")
sys.exit(1 if FAILED else 0)
