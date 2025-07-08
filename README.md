# 🗳️ VotingBlockchain

**VotingBlockchain** is a blockchain implementation in Go, based on Bitcoin, and tailored for decentralized voting. It includes full-node functionality, block mining, peer-to-peer networking, transaction verification, and signature validation — all built with a voting system in mind.

---

## 🏃 How to Run

1. Open a terminal in the **project root directory**.
2. Run the following command:

```bash
go run ./cmd/main/
```

> Make sure you’re in the project root — not inside `cmd/main` — so paths like `config/config.yml` and `databases/database.db` resolve correctly.

---

## ⚙️ Configuration File

The application uses a YAML config file at `config/config.yml` by default. You can override the path using the `CONFIG_FILE` environment variable.

### Example: `config/config.yml`

```yaml
network:
  ip: 0.0.0.0
  port: 8333
  ping-interval: 120
  pong-timeout: 1200
  send-data-interval: 100

node:
  version: 1
  type: 1

government:
  public-key: "HEX_ENCODED_GOVERNMENT_PUBLIC_KEY"

miner:
  enabled: true
  public-key: "HEX_ENCODED_YOUR_MINER_PUBLIC_KEY"
```

### Key Fields:
- `node.type`: Type of node to run. **Currently only `full(1)` is supported.**
- `government.public-key`: The trusted base64-encoded public key used to verify signed votes or actions from the government.
- `miner.public-key`: Your node’s base64-encoded public key used to sign your own transactions.

---

## 🗃️ Database

- The blockchain uses **SQLite** with the **GORM** ORM for data storage.
- The default database path is `databases/blockchain.db`.
- You can override it with the `DATABASE_FILE` environment variable.

### Requirements:
- **CGO must be enabled**
- **GCC must be installed** (required by SQLite driver used in Go)

> On Windows, use MinGW or install TDM-GCC.  
> On Linux/macOS, use your system's package manager to install `gcc`.

---

## 🧪 Running Tests

To run all tests:

```bash
go test ./tests/...
```

To run specific test packages, like the full node tests:

```bash
go test ./tests/internal/nodes/ -v
```

> During tests, the mining difficulty is set very low (`0x207fffff`) so blocks are mined quickly.

---

## 🔧 Go Version

This project uses **Go 1.23.4**.  
Make sure you are using this version (or newer) for compatibility.

---

## ✅ Summary

- Built in Go, inspired by Bitcoin
- Designed for decentralized voting use cases
- Full-node mining, networking, and persistence
- Configurable via YAML
- SQLite storage with GORM
- CGO and GCC are required for database support
- Supports unit and integration tests
