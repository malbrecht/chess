package chess

type movegen struct {
	*Board
	moves []Move
}

// LegalMoves returns the list of moves that can be played in this position.
func (b *Board) LegalMoves() []Move {
	moves, _ := b.pseudoLegalMoves()
	j := 0
	for i := 0; i < len(moves); i++ {
		if moves[i].isLegal(b) {
			moves[i] = moves[j]
			j++
		}
	}
	return moves[:j]
}

// pseudoLegalMoves returns the list of "pseudo-legal" moves in the current
// position (i.e. moves that are legal except that they may leave one's own
// king in check). Returns (nil, true) if the position is illegal because the
// opponent's king is in check.
func (b *Board) pseudoLegalMoves() (moves []Move, check bool) {
	gen := movegen{Board: b}
	for i, piece := range gen.Piece {
		if piece == NoPiece || piece.Color() != gen.SideToMove {
			continue
		}
		sq := Sq(i)
		switch piece.Type() {
		case Pawn:
			gen.pawn(sq)
		case Knight:
			gen.knight(sq)
		case Bishop:
			gen.bishop(sq)
		case Rook:
			gen.rook(sq)
		case Queen:
			gen.bishop(sq)
			gen.rook(sq)
		case King:
			gen.king(sq)
		}
	}
	// the position is illegal if the opponent is in check
	checkFrom, checkTo := gen.checkFrom, gen.checkTo
	if checkFrom == A1 && checkTo == A1 {
		checkFrom = gen.find(b.opp(King), A1, H8)
		checkTo = checkFrom
	}
	for _, move := range gen.moves {
		if move.To >= checkFrom && move.To <= checkTo {
			return nil, true
		}
	}
	return gen.moves, false
}

// step returns the square reached by a piece stepping the given offset. It
// returns NoSquare if the piece would fall off the board. The offset must not
// jump more than two files (a knight's jump) because jumps >2 files are used
// to detect warps around the board.
func (from Sq) step(offset int) Sq {
	to := from + Sq(offset)
	if to < A1 || to > H8 {
		return NoSquare
	}
	if dx := to.File() - from.File(); dx < -2 || dx > 2 {
		return NoSquare
	}
	return to
}

// addMove adds a move if the to square is on the board and the move is not
// blocked by a friendly piece. Returns whether the piece can move on.
func (gen *movegen) addMove(from, to Sq, promotion Piece) bool {
	if to == NoSquare {
		return false
	}
	blocker := gen.Piece[to]
	if blocker == NoPiece || blocker.Color() != gen.SideToMove {
		gen.moves = append(gen.moves, Move{from, to, promotion})
	}
	return blocker == NoPiece
}

// Pawns

func (gen *movegen) pawn(sq Sq) {
	offset := []int{8, -8}[gen.SideToMove]
	ok := gen.pawnPush(sq, sq.step(offset))
	if ok && sq.RelativeRank(gen.SideToMove) == Rank2 {
		gen.pawnPush(sq, sq.step(2*offset))
	}
	gen.pawnCapture(sq, sq.step(offset+1))
	gen.pawnCapture(sq, sq.step(offset-1))
}

func (gen *movegen) pawnPush(from, to Sq) bool {
	if to != NoSquare && gen.Piece[to] == NoPiece {
		return gen.addPawnMove(from, to)
	}
	return false
}

func (gen *movegen) pawnCapture(from, to Sq) {
	if to != NoSquare && (gen.Piece[to] != NoPiece || to == gen.EpSquare) {
		gen.addPawnMove(from, to)
	}
}

func (gen *movegen) addPawnMove(from, to Sq) bool {
	if to.RelativeRank(gen.SideToMove) == Rank8 {
		gen.addMove(from, to, gen.my(Knight))
		gen.addMove(from, to, gen.my(Bishop))
		gen.addMove(from, to, gen.my(Rook))
		gen.addMove(from, to, gen.my(Queen))
		return false
	}
	return gen.addMove(from, to, NoPiece)
}

// Knights

func (gen *movegen) knight(sq Sq) {
	for _, offset := range []int{-17, -15, -10, -6, 6, 10, 15, 17} {
		gen.addMove(sq, sq.step(offset), NoPiece)
	}
}

// Bishops and rooks (sliders)

func (gen *movegen) slider(from Sq, offset int) {
	to := from.step(offset)
	for gen.addMove(from, to, NoPiece) {
		to = to.step(offset)
	}
}

func (gen *movegen) bishop(from Sq) {
	for _, offset := range []int{-9, -7, 7, 9} {
		gen.slider(from, offset)
	}
}

func (gen *movegen) rook(from Sq) {
	for _, offset := range []int{-8, -1, 1, 8} {
		gen.slider(from, offset)
	}
}

// King

func (gen *movegen) king(from Sq) {
	for _, offset := range []int{-9, -8, -7, -1, 1, 7, 8, 9} {
		gen.addMove(from, from.step(offset), NoPiece)
	}
	if gen.canCastle(kingSide) {
		to := gen.CastleSq[gen.SideToMove|kingSide]
		gen.moves = append(gen.moves, Move{From: from, To: to})
	}
	if gen.canCastle(queenSide) {
		to := gen.CastleSq[gen.SideToMove|queenSide]
		gen.moves = append(gen.moves, Move{From: from, To: to})
	}
}

// castleSquares returns the king move (kf->kt) and rook move (rf->rt) for a
// castling move, as well as the smallest range [min,max] of squares that
// contains all of the first four squares. Returns rf=NoSquare if castling is
// not allowed.
func (b *Board) castleSquares(wing int) (rf, kf, rt, kt, min, max Sq) {
	rf = b.CastleSq[b.SideToMove|wing]
	if rf == NoSquare {
		return
	}
	kf = b.find(b.my(King), A1, H8)
	rt = []Sq{D1, D8, F1, F8}[b.SideToMove|wing]
	kt = []Sq{C1, C8, G1, G8}[b.SideToMove|wing]

	min, max = H8, A1
	for _, sq := range []Sq{kf, rf, kt, rt} {
		if sq < min {
			min = sq
		}
		if sq > max {
			max = sq
		}
	}
	return
}

// canCastle returns whether the side to move can castle on the given wing.
// Note: this does not check whether the king moves through an attacked square;
// use move.isLegal() for that.
func (b *Board) canCastle(wing int) bool {
	rf, kf, _, _, min, max := b.castleSquares(wing)
	if rf == NoSquare {
		return false
	}
	// cannot castle if there are other pieces in the [min,max] range
	for sq := min; sq <= max; sq++ {
		if b.Piece[sq] != NoPiece && sq != kf && sq != rf {
			return false
		}
	}
	return true
}

// IsCheckOrMate returns whether the side to move is in check and/or has been
// mated. Mate without check means stalemate.
func (b *Board) IsCheckOrMate() (check, mate bool) {
	_, check = b.MakeMove(NullMove).pseudoLegalMoves()

	moves, _ := b.pseudoLegalMoves()
	for _, move := range moves {
		if move.isLegal(b) {
			mate = false // at least one move: not mate
			return
		}
	}
	mate = true // no moves: mate or stalemate
	return
}
