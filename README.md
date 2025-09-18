# 🗳️ VotingBlockchain

**VotingBlockchain** is a blockchain implementation in Go, inspired by Bitcoin and tailored for decentralized voting. It includes full‑node functionality, block mining, peer‑to‑peer networking, transaction verification, and signature validation — all built with a voting system in mind.

---

## 🏃 How to Run

1. Open a terminal in the **project root directory**.
2. Run:

```bash
go run ./cmd/main/
```

> Make sure you’re in the project root — not inside `cmd/main` — so relative paths like `config/config.yml` and `databases/...` resolve correctly.

---

## 📦 Go Modules

VotingBlockchain uses **Go modules** for dependency management. To download and install all required modules, run the following command in the **project root**:

```bash
go mod download
```

This will:

- Download all dependencies specified in `go.mod`
- Ensure the project builds correctly with all required packages

> Make sure your Go version matches the project’s requirement (Go 1.23.4 or newer).

---

## ⚙️ Configuration

By default the app reads `config/config.yml`. You can override the path using the `CONFIG_FILE` environment variable.

### Example: `config/config.yml`

```yaml
node:
  version: 1
  type: 1  # 1 = full node

network:
  ip: 127.0.0.1
  port: 8333
  ping-interval: 120
  pong-timeout: 1200
  send-data-interval: 100
  get-addr-interval: 300
  max-number-of-connections: 10
  addresses-file: "addresses/addresses.json"
  dial: true

miner:
  enabled: true
  public-key: "HEX_ENCODED_YOUR_MINER_PUBLIC_KEY"  # compressed secp256k1, hex

government:
  public-key: "HEX_ENCODED_GOVERNMENT_PUBLIC_KEY"  # compressed secp256k1, hex

ui:
  enabled: true  # enables the built-in UI for voting & monitoring

database:
  file: "databases/blockchain-test.db"  # path to SQLite database file

voters:
  file: "voters/voters.json"  # path to pre-generated voters for the UI
```

### Key fields

* `node.type`: Node role. **Currently only `full (1)` is supported.**
* `network.*`: P2P settings (bind IP/port and timing intervals).
* `miner.enabled`: Turns the miner on/off.
* `miner.public-key`: Your miner’s **hex-encoded compressed** secp256k1 public key.
* `government.public-key`: The trusted **hex-encoded compressed** secp256k1 public key used to verify government signatures.
* `ui.enabled`: Enables the built-in graphical UI for casting votes and monitoring blocks/transactions.
* `database.file`: SQLite file path for blockchain state.
* `voters.file`: Path to a JSON file containing voters (used by the UI to create valid, signed vote transactions).
* `network.addresses-file`: Path to a json file containing addresses

---

## 🖥️ User Interface (UI)

The VotingBlockchain project includes a **built-in desktop UI** built with [Fyne](https://fyne.io/).  
This interface allows you to:

- View the list of registered voters from the configured `voters.json` file
- Select a voter and cast votes through the blockchain network
- Monitor blockchain status, including mined blocks and transaction history
- Interact with the node without needing to use CLI commands

### Enabling the UI
The UI can be enabled in the config file:

```yaml
ui:
  enabled: true
```

When enabled, the UI launches alongside your node when you run:

```bash
go run ./cmd/main/
```

---

## 👥 Voters JSON (used by the UI)

The UI can take in a JSON file that lists voters and includes a **government signature** per voter. Each government signature is an ECDSA signature created by the government **over the hash of the voter’s public key bytes**.

### Format

```json
[
  {
    "name": "Voter1",
    "government_signature": "DER_HEX_SIGNATURE",
    "private_key": "HEX_ENCODED_PRIVATE_KEY_DER",
    "public_key": "HEX_ENCODED_COMPRESSED_PUBLIC_KEY"
  },
  {
    "name": "Voter2",
    "government_signature": "DER_HEX_SIGNATURE",
    "private_key": "HEX_ENCODED_PRIVATE_KEY_DER",
    "public_key": "HEX_ENCODED_COMPRESSED_PUBLIC_KEY"
  }
]
```

**Notes**

* `public_key` is the voter’s **compressed** secp256k1 public key (hex).
* `private_key` is the voter’s private key in **DER**, hex‑encoded (used only locally by the UI to sign the transaction).
* `government_signature` is a **DER** ECDSA signature, hex‑encoded, produced by the government’s private key over `hash(voter_public_key_bytes)`.

> The node validates each transaction by checking the government’s signature against the configured `government.public-key`.

---

## 🌐 Addresses JSON

The node can take in a JSON file that lists addresses to include in the database and connect to.

### Format

```json
[
  {
    "ip": "127.0.0.1",
    "port": 8333,
    "node_type": 1
  }
  {
    "ip": "127.0.0.1",
    "port": 8334,
    "node_type": 1
  }
]
```

---

## 🗃️ Database

* Uses **SQLite** with **GORM**.
* The database path is configured via `database.file` (see `config/config.yml`).

### Requirements

* **CGO must be enabled**
* **GCC must be installed** (required by the SQLite driver)

> Windows: install MinGW/TDM-GCC.
> Linux/macOS: install `gcc` via your package manager.

---

## 🧪 Running Tests

Run all tests:

```bash
go test ./tests/...
```

Run a specific package (e.g., full node tests):

```bash
go test ./tests/internal/nodes/ -v
```

> During tests, the mining difficulty is set very low (e.g., `0x207fffff`) so blocks are mined quickly.

---

## 🔧 Go Version

Use **Go 1.23.4** or newer.

---

## ✅ Summary

* Built in Go, inspired by Bitcoin
* Full-node mining, networking, and persistence
* UI mode for voting and blockchain monitoring
* Voters JSON support for UI-driven vote casting
* Configurable via YAML
* SQLite storage with GORM (CGO + GCC required)
* Unit and integration tests supported
