# Game architectural modelling

This document is an initial set of thoughts about how the game should be 
modelled for implementation. We will need to refine this as we build.

As we design and implement pieces, we will create specs under `docs/specs` for
relevant pieces. 

## Core flow

### Players

A player accesses the game via their web browser. A player should be able to select a display name and play unregistered, but we should also have a way for them to register an account with an username and password if they wish, allowing for storage of stats or game history. A registered player may have a changeable display name separate to their login username.

We follow best practice for storage of player data, including securely storing their password. While not in initial requirements, we should design such that we could integrate with other login providers (e.g. Google) via OAuth down the line.

### Lobbies

When a player (logged in or unregistered) enters the webapp, they may start a new lobby, or join an existing lobby by its lobby code.

A lobby represents a set of players (where players may join/leave the lobby at any time) which can then play multiple games in sequence. A lobby is identified by a code (ideally, a human readable code so players can share it easily, but which provides sufficient uniqueness). Lobbies are transient: if all players leave a lobby it no longer exists. 

The lobby's creator is the initial host of the lobby, and they may make another player the host. If the host leaves, a new host is automatically assigned.

The lobby has a history of games played and their outcomes, and a game may be active or not at a given time. The host has the power to start a new game, when one is not in progress, and to abandon a game in progress.

A lobby may have spectators in addition to active players. If a new player joins the lobby during a game, they will initially be a spectator. The host may promote spectators to players when a game is not in progress. A player may switch between spectator/player when a game is not in progress.

The host may themselves be a player or a spectator.

Generally, during a game, players will have only the information relevant to them about the game state, but spectators may see the boards of all players.

The lobby controller's ultimate job is to manage the state machine of the whole lobby - handling events such as players joining/leaving, swapping to/from spectators, games starting/completing/being abandoned etc.

Players also interact with the actual game _through_ the lobby, although the lobby controller does not model the state machine of a single ongoing game. There is overlap since a game is impacted by lobby events if e.g. an active player leaves.

### Games

A game is an individual instance of the crossword game, bound to a lobby. It has a list of players (and doesn't specifically care about spectators, since they do not affect the game logic).

The game controller manages the state machine of the actual game: handling the initial setup, the taking of turns by each player through announcement -> placement actions, and the handling of scoring once the game is complete.

### Boards

The board is the data model representing an individual player's board grid, starting empty. During each game turn, the announcing player will declare a letter, and then each player will be able to select an empty square to place it in. The game is over when the player grids are full.