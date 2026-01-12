#!/bin/bash
#
# Simulate another player in a crossword game
#
# Usage: ./scripts/simulate-player.sh <lobby-code> [player-name]
#
# This script:
# 1. Creates a guest player
# 2. Joins the specified lobby
# 3. Plays automatically (announces letters when announcer, places at next position)
#

set -e

LOBBY_CODE="${1:-}"
PLAYER_NAME="${2:-Bot}"
SERVER="${CWGAME_SERVER:-http://localhost:8080}"

if [ -z "$LOBBY_CODE" ]; then
    echo "Usage: $0 <lobby-code> [player-name]"
    exit 1
fi

# Find project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Build CLI if needed
CLI="$PROJECT_ROOT/bin/cwgame"
if [ ! -f "$CLI" ]; then
    echo "Building CLI..."
    cd "$PROJECT_ROOT" && go build -o bin/cwgame ./cmd/cwgame
fi

# Create a temp token file for this player
TOKEN_FILE=$(mktemp)
trap "rm -f $TOKEN_FILE" EXIT

# Helper function to run CLI
run_cli() {
    "$CLI" --server "$SERVER" --token-file "$TOKEN_FILE" "$@"
}

run_cli_json() {
    "$CLI" --server "$SERVER" --token-file "$TOKEN_FILE" --output json "$@"
}

echo "=== Simulated Player: $PLAYER_NAME ==="
echo "Server: $SERVER"
echo "Lobby: $LOBBY_CODE"
echo ""

# Create guest player
echo "Creating guest player..."
AUTH_RESP=$(run_cli_json player guest --name "$PLAYER_NAME")
PLAYER_ID=$(echo "$AUTH_RESP" | jq -r '.player.id')
echo "Player ID: $PLAYER_ID"

# Join lobby
echo "Joining lobby $LOBBY_CODE..."
LOBBY_RESP=$(run_cli_json lobby join "$LOBBY_CODE")
GRID_SIZE=$(echo "$LOBBY_RESP" | jq -r '.config.grid_size')
echo "Joined! Grid size: $GRID_SIZE"

# Track placement position
NEXT_ROW=0
NEXT_COL=0

# Letters to use when announcing (cycle through)
LETTERS=(A B C D E F G H I J K L M N O P Q R S T U V W X Y Z)
LETTER_INDEX=0

advance_position() {
    NEXT_COL=$((NEXT_COL + 1))
    if [ $NEXT_COL -ge $GRID_SIZE ]; then
        NEXT_COL=0
        NEXT_ROW=$((NEXT_ROW + 1))
    fi
}

get_next_letter() {
    LETTER="${LETTERS[$LETTER_INDEX]}"
    LETTER_INDEX=$(( (LETTER_INDEX + 1) % ${#LETTERS[@]} ))
    echo "$LETTER"
}

echo ""
echo "=== Starting game loop ==="
echo "Waiting for game events..."
echo ""

# Listen for events and respond
while true; do
    # Get current game state
    GAME_RESP=$(run_cli_json game get "$LOBBY_CODE" 2>/dev/null || echo '{"state":"none"}')
    GAME_STATE=$(echo "$GAME_RESP" | jq -r '.state // "none"')

    if [ "$GAME_STATE" = "none" ] || [ "$GAME_STATE" = "null" ]; then
        echo "[$(date +%H:%M:%S)] No active game, waiting..."
        sleep 2
        continue
    fi

    if [ "$GAME_STATE" = "scoring" ]; then
        echo "[$(date +%H:%M:%S)] Game complete!"
        echo "$GAME_RESP" | jq '.scores'
        break
    fi

    if [ "$GAME_STATE" = "abandoned" ]; then
        echo "[$(date +%H:%M:%S)] Game abandoned."
        break
    fi

    CURRENT_ANNOUNCER=$(echo "$GAME_RESP" | jq -r '.current_announcer')
    CURRENT_TURN=$(echo "$GAME_RESP" | jq -r '.current_turn')

    if [ "$GAME_STATE" = "announcing" ]; then
        if [ "$CURRENT_ANNOUNCER" = "$PLAYER_ID" ]; then
            LETTER=$(get_next_letter)
            echo "[$(date +%H:%M:%S)] Turn $CURRENT_TURN: I'm announcer! Announcing '$LETTER'..."
            run_cli_json game announce "$LOBBY_CODE" "$LETTER" > /dev/null
        else
            echo "[$(date +%H:%M:%S)] Turn $CURRENT_TURN: Waiting for announcer..."
            sleep 1
        fi
        continue
    fi

    if [ "$GAME_STATE" = "placing" ]; then
        # Check if we've already placed
        MY_PLACED=$(echo "$GAME_RESP" | jq -r ".placements[\"$PLAYER_ID\"] // false")

        if [ "$MY_PLACED" = "true" ]; then
            echo "[$(date +%H:%M:%S)] Turn $CURRENT_TURN: Already placed, waiting for others..."
            sleep 1
            continue
        fi

        CURRENT_LETTER=$(echo "$GAME_RESP" | jq -r '.current_letter')
        echo "[$(date +%H:%M:%S)] Turn $CURRENT_TURN: Placing '$CURRENT_LETTER' at ($NEXT_ROW, $NEXT_COL)..."

        PLACE_RESP=$(run_cli_json game place "$LOBBY_CODE" "$NEXT_ROW" "$NEXT_COL")
        TURN_COMPLETE=$(echo "$PLACE_RESP" | jq -r '.turn_complete')
        GAME_COMPLETE=$(echo "$PLACE_RESP" | jq -r '.game_complete')

        advance_position

        if [ "$GAME_COMPLETE" = "true" ]; then
            echo "[$(date +%H:%M:%S)] Game complete!"
            WINNER=$(echo "$PLACE_RESP" | jq -r '.winner // "tie"')
            echo "Winner: $WINNER"
            break
        fi

        if [ "$TURN_COMPLETE" = "true" ]; then
            echo "[$(date +%H:%M:%S)] Turn complete!"
        fi

        continue
    fi

    echo "[$(date +%H:%M:%S)] Unknown state: $GAME_STATE"
    sleep 1
done

echo ""
echo "=== Bot finished ==="
