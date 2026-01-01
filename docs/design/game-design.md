# 'Crossword game' overview

The idea here is an implementation of a simple word game I played a bunch with
my family as a kid. Never had a name for it other than as a 'crossword game',
although it is not a _crossword_, clearly. I should probably come up with
something better.

This document describes the game in general. Other docs discuss how we will implement the game more specifically.

## Overview of the game

The crossword game is played by players (usually 2-4 players) on individual grids, generally of 5x5 although it should be customisable, which start empty.

Each player takes turns in announcing a letter, which all players must then 
place in an empty square of their grid.

Once the grid is full, the players score based on words they have created either
vertically top-to-bottom or horizontally left-to-right in the grid. The scoring
rules are:

* Each letter may only be used once in each direction
  * i.e. the same letter may be used for a horizontal and a vertical word, but
    not in two overlapping words in the same direction
* Words must be valid dictionary words of at least two letters in length
* Words score their length
* Words which score the full length of a row/column score double their length (i.e. 10 points)

Depending on grid size, players will generally not get an equal number
of turns. For now at least this unfairness is accepted.

