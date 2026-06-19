# Game of the Amazons - Bot Tournament Server

Welcome to the **Game of the Amazons Tournament Server**. This repository contains the core game engine, automated matchmaking queue, and gRPC communication layer designed to host headless bot-vs-bot competitions.

This platform allows developers to implement their game-playing bots in any programming language of their choice, connecting seamlessly to a central server via gRPC to compete in automated, high-performance tournaments.

---

## 🎮 What is the Game of the Amazons?

The **Game of the Amazons** (originally *El Juego de las Amazonas*) is a two-player, abstract strategy game invented by Walter Zamkauskas in 1988. It combines the movement mechanics of Chess with the territorial isolation elements of Go.

### The Components

* **The Board:** A $10 \times 10$ checkerboard grid.
* **The Pieces:** Each player (White and Black) controls **4 Amazons**.
* **The Arrows:** "Burned" squares that permanently block movement and sightlines as the game progresses.

### Turn Mechanics

Players alternate turns, with White moving first. A single turn is strictly divided into two sequential phases:

1. **Move an Amazon:** The player chooses one of their 4 Amazons and moves her like a **Chess Queen** (any number of empty squares orthogonally or diagonally). She cannot jump over or land on other Amazons or burned squares.
2. **Shoot an Arrow:** From the square where the Amazon just landed, she must shoot an "arrow" in any legal Queen-move direction. The square where the arrow lands becomes permanently **burned** (often marked as an 'X' or an arrow on the board).

> ⚠️ **Note:** An Amazon can shoot an arrow back through the square she just vacated, but arrows cannot cross or land on occupied or already burned squares.

### Winning Condition

As the board fills up with arrows, the grid becomes fragmented into isolated pockets.

* The game ends when a player **cannot make any legal moves** on their turn (i.e., all 4 of their Amazons are trapped by burned squares or other pieces).
* The last player to make a successful legal move **wins**.

---

## 🛠️ Project Architecture

This server is built using **Go (Golang)** for high-concurrency performance and **gRPC** for language-agnostic API streaming.

```text
├── api/proto/         # The Protocol Buffers (.proto) contract file
├── cmd/server/        # Entry point; initializes configs and launches gRPC listener
├── configs/           # Environment and server configuration variables
├── internal/
│   ├── game/          # Amazons game engine, board state, and move validation
│   ├── matchmaking/   # Concurrency-safe pairing queue for active bots
│   └── server/        # gRPC handlers and stream managers
└── go.mod             # Go module dependencies
