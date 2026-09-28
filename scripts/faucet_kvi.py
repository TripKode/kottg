#!/usr/bin/env python3
# KSP-1: Faucet KVI — acuña Kodechain Value Instruments on-chain.
# Migración del faucet: en vez de dispensar KDC nativo, este faucet es el
# minter KVI-M de un contrato ValueInstrument desplegado en la testnet.
#
# Uso:
#   python3 faucet_kvi.py 0x<dirección>     → acuña 1000 KVI (cooldown 1h)
#   python3 faucet_kvi.py --status          → estado del faucet y cooldowns
#
# El estado (contrato + cooldowns) persiste en kvi_faucet_state.json.
import json, urllib.request, subprocess, sys, time, os

ENGINE   = "http://localhost:8084"
MINTER   = "0x66d1acb3f9065eabecd97b0c2df07de45245518f"  # faucet testnet
AMOUNT   = 1000      # KVI por solicitud
COOLDOWN = 3600       # 1 hora por destinatario
STATE    = os.path.join(os.path.dirname(os.path.abspath(__file__)), "kvi_faucet_state.json")

# KSP-6B: selectores KSEL-8 computados en runtime desde la firma canónica.
_SEL_CACHE = {}
def sel(sig):
    if sig not in _SEL_CACHE:
        out = subprocess.run(["/tmp/kselbin", sig], capture_output=True, text=True)
        _SEL_CACHE[sig] = out.stdout.split()[1]
    return _SEL_CACHE[sig]

SEL = {"init": sel("init(address)"), "mint": sel("mint(address,u256)"),
       "balanceOf": sel("balanceOf(address)"), "totalSupply": sel("totalSupply()")}

def ident(a):
    return int(subprocess.run(["/tmp/kselbin", a], capture_output=True, text=True).stdout.split()[-1], 16)

def post(path, payload):
    r = urllib.request.Request(ENGINE + path, data=json.dumps(payload).encode(),
                               headers={"Content-Type": "application/json"})
    return json.load(urllib.request.urlopen(r, timeout=30))

def call(addr, sel, *args, caller=MINTER):
    def word(a):
        return format((int(a, 16) if isinstance(a, str) else a) & (2**256-1), "064x")
    inp = sel + "00"*28 + "".join(word(a) for a in args)
    ex = post("/api/contract/call", {"contract_address": addr, "caller": caller,
        "input": inp, "gas_limit": 4000000, "gas_price": 1000000000}).get("execution", {})
    if not ex.get("success"):
        raise RuntimeError("llamada fallida: " + str(ex.get("error", "")))
    return ex

def w(rd, i=0):
    return int(rd[2 + 64*i: 2 + 64*(i+1)], 16)

def load_state():
    if os.path.exists(STATE):
        with open(STATE) as f:
            return json.load(f)
    return {"contract": None, "cooldowns": {}}

def save_state(st):
    with open(STATE, "w") as f:
        json.dump(st, f, indent=2)

def ensure_contract(st):
    """Despliega + inicializa el KVI del faucet si no existe (idempotente)."""
    if st.get("contract"):
        return st["contract"]
    bytecode = open(os.path.join(os.path.dirname(STATE), "..", "contracts",
                                 "ValueInstrument.bin"), "rb").read().hex()
    resp = post("/api/contract/deploy", {"creator": MINTER, "bytecode": bytecode,
                                         "name": "KVI-Faucet", "gasLimit": 8000000})
    addr = resp["contract"]["address"]
    call(addr, SEL["init"], ident(MINTER))          # minter = faucet
    st["contract"] = addr
    save_state(st)
    print(f"🚰 KVI faucet desplegado e inicializado: {addr}")
    return addr

def main():
    st = load_state()
    if len(sys.argv) >= 2 and sys.argv[1] == "--status":
        addr = st.get("contract") or "(sin desplegar)"
        print(f"Contrato KVI: {addr}")
        print(f"Minter:       {MINTER}")
        if st.get("contract"):
            print(f"Total acuñado: {w(call(st['contract'], SEL['totalSupply'])['return_data'])} KVI")
        for r, ts in st.get("cooldowns", {}).items():
            left = int(ts + COOLDOWN - time.time())
            print(f"  {r}: {'disponible' if left <= 0 else f'cooldown {left}s'}")
        return 0

    if len(sys.argv) != 2 or not sys.argv[1].startswith("0x") or len(sys.argv[1]) != 42:
        print("uso: faucet_kvi.py 0x<dirección de 40 hex> | --status")
        return 1
    recipient = sys.argv[1].lower()

    addr = ensure_contract(st)
    now = time.time()
    last = st["cooldowns"].get(recipient, 0)
    if now - last < COOLDOWN:
        left = int(COOLDOWN - (now - last))
        print(f"⏳ Cooldown activo para {recipient}: {left}s restantes")
        return 1

    call(addr, SEL["mint"], ident(recipient), AMOUNT)
    st["cooldowns"][recipient] = now
    save_state(st)

    bal = w(call(addr, SEL["balanceOf"], ident(recipient))["return_data"])
    sup = w(call(addr, SEL["totalSupply"])["return_data"])
    print(f"✅ Acuñados {AMOUNT} KVI para {recipient}")
    print(f"   Saldo on-chain: {bal} KVI | Oferta total: {sup} KVI")
    return 0

if __name__ == "__main__":
    sys.exit(main())
