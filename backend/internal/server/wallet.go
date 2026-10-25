package server

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

// charge deducts from the user's wallet inside tx, failing with 402 if the balance is too low.
func charge(ctx context.Context, tx pgx.Tx, userID, amount int64, reason, ref string) error {
	if amount <= 0 {
		return nil
	}
	tag, err := tx.Exec(ctx, `UPDATE users SET balance_tetri = balance_tetri - $2 WHERE id = $1 AND balance_tetri >= $2`, userID, amount)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errStatus(http.StatusPaymentRequired, "Not enough balance. Top up your wallet first.")
	}
	_, err = tx.Exec(ctx, `INSERT INTO wallet_transactions (user_id, amount_tetri, reason, reference) VALUES ($1, $2, $3, $4)`,
		userID, -amount, reason, ref)
	return err
}

type WalletTx struct {
	AmountTetri int64     `json:"amountTetri"`
	Reason      string    `json:"reason"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (s *Server) wallet(w http.ResponseWriter, r *http.Request, u *User) error {
	rows, err := s.db.Query(r.Context(), `
		SELECT amount_tetri, reason, created_at FROM wallet_transactions WHERE user_id = $1 ORDER BY id DESC LIMIT 50`, u.ID)
	if err != nil {
		return err
	}
	txs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (WalletTx, error) {
		var t WalletTx
		return t, row.Scan(&t.AmountTetri, &t.Reason, &t.CreatedAt)
	})
	if err != nil {
		return err
	}
	var boost int64
	if err := s.db.QueryRow(r.Context(), `SELECT COALESCE(SUM(amount_tetri), 0)::bigint FROM boosts WHERE user_id = $1 AND ends_at > now()`, u.ID).Scan(&boost); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": u, "activeBoostTetri": boost, "transactions": txs})
	return nil
}

// topUp adds money to the wallet. Only the "mock" payments mode exists until a real payment provider is connected.
func (s *Server) topUp(w http.ResponseWriter, r *http.Request, u *User) error {
	if s.cfg.PaymentsMode != "mock" {
		return errStatus(http.StatusForbidden, "Online payments are not connected yet")
	}
	var in struct {
		AmountTetri int64 `json:"amountTetri"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return err
	}
	if in.AmountTetri < 100 || in.AmountTetri > 100_000 {
		return badRequest("Top up between 1 and 1000 GEL")
	}
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(r.Context(), `UPDATE users SET balance_tetri = balance_tetri + $2 WHERE id = $1`, u.ID, in.AmountTetri); err != nil {
			return err
		}
		_, err := tx.Exec(r.Context(), `INSERT INTO wallet_transactions (user_id, amount_tetri, reason) VALUES ($1, $2, 'Top up (test payment)')`,
			u.ID, in.AmountTetri)
		return err
	})
	if err != nil {
		return err
	}
	return s.reloadUser(w, r, u.ID)
}

func (s *Server) reloadUser(w http.ResponseWriter, r *http.Request, id int64) error {
	u, err := scanUser(s.db.QueryRow(r.Context(), `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, u)
	return nil
}

func (s *Server) payRegistration(w http.ResponseWriter, r *http.Request, u *User) error {
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		if err := charge(r.Context(), tx, u.ID, s.cfg.RegistrationFee, "Registration fee ("+strconv.Itoa(s.cfg.RegistrationDays)+" days)", ""); err != nil {
			return err
		}
		_, err := tx.Exec(r.Context(), `
			UPDATE users SET registration_paid_until = GREATEST(COALESCE(registration_paid_until, now()), now()) + make_interval(days => $2)
			WHERE id = $1`, u.ID, s.cfg.RegistrationDays)
		return err
	})
	if err != nil {
		return err
	}
	return s.reloadUser(w, r, u.ID)
}

func (s *Server) buyBoost(w http.ResponseWriter, r *http.Request, u *User) error {
	if s.cfg.PaymentsMode == "free" {
		return errStatus(http.StatusForbidden, "Promoted placement opens when online payments launch")
	}
	var in struct {
		AmountTetri int64 `json:"amountTetri"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return err
	}
	if in.AmountTetri < 100 || in.AmountTetri > 1_000_000 {
		return badRequest("Boost between 1 and 10000 GEL")
	}
	if !u.RegistrationActive {
		return errStatus(http.StatusPaymentRequired, "Pay the registration fee before boosting")
	}
	err := pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		if err := charge(r.Context(), tx, u.ID, in.AmountTetri, "Leaderboard boost ("+strconv.Itoa(s.cfg.BoostDays)+" days)", ""); err != nil {
			return err
		}
		_, err := tx.Exec(r.Context(), `INSERT INTO boosts (user_id, amount_tetri, ends_at) VALUES ($1, $2, now() + make_interval(days => $3))`,
			u.ID, in.AmountTetri, s.cfg.BoostDays)
		return err
	})
	if err != nil {
		return err
	}
	return s.leaderboardFor(w, r, u.Role)
}

type LeaderboardEntry struct {
	Rank       int    `json:"rank"`
	UserID     int64  `json:"userId"`
	Name       string `json:"name"`
	Detail     string `json:"detail"`
	BoostTetri int64  `json:"boostTetri"`
}

func (s *Server) leaderboard(w http.ResponseWriter, r *http.Request) error {
	role := r.URL.Query().Get("role")
	if role != "technician" && role != "seller" {
		return badRequest("role must be technician or seller")
	}
	return s.leaderboardFor(w, r, role)
}

// leaderboardFor ranks users by the sum of their active boosts; paying more than the current top takes first place.
func (s *Server) leaderboardFor(w http.ResponseWriter, r *http.Request, role string) error {
	rows, err := s.db.Query(r.Context(), `
		SELECT u.id, u.name,
		       COALESCE(sp.shop_name || COALESCE(' · stall ' || st.code, ''),
		                (SELECT string_agg(c.name, ', ' ORDER BY c.name) FROM technician_categories tc
		                 JOIN categories c ON c.id = tc.category_id WHERE tc.user_id = u.id), ''),
		       SUM(b.amount_tetri)::bigint AS total
		FROM boosts b
		JOIN users u ON u.id = b.user_id
		LEFT JOIN seller_profiles sp ON sp.user_id = u.id AND u.role = 'seller'
		LEFT JOIN stalls st ON st.id = sp.stall_id
		WHERE b.ends_at > now() AND u.role = $1 AND u.registration_paid_until > now()
		GROUP BY u.id, u.name, sp.shop_name, st.code
		ORDER BY total DESC, MIN(b.starts_at) ASC
		LIMIT 20`, role)
	if err != nil {
		return err
	}
	entries, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (LeaderboardEntry, error) {
		var e LeaderboardEntry
		return e, row.Scan(&e.UserID, &e.Name, &e.Detail, &e.BoostTetri)
	})
	if err != nil {
		return err
	}
	for i := range entries {
		entries[i].Rank = i + 1
	}
	writeJSON(w, http.StatusOK, map[string]any{"role": role, "entries": entries, "boostDays": s.cfg.BoostDays})
	return nil
}
